// musicplayerd is a headless music player daemon that connects to musicd as a
// remote-controlled player. It uses mpv for audio playback and optionally streams
// audio via RTP/HTTP using FFmpeg.
//
// Environment variables:
//
//	MUSICD_URL       - URL of the musicd server (required, e.g. http://localhost:8080)
//	SESSION_NAME     - session display name (default: {hostname}-player)
//	STATE_DIR        - directory for persisting session ID (default: /var/lib/musicplayerd)
//	AUDIO_DEVICE     - PulseAudio sink name (default: auto)
//	AUDIO_VOLUME     - initial volume 0-100 (default: 80)
//	STREAM_ENABLED   - enable streaming outputs (default: false)
//	STREAM_RTP_DEST  - RTP destination (e.g. rtp://239.0.0.1:5004)
//	STREAM_HTTP_PORT - HTTP stream listen port (e.g. 8090)
//	STREAM_FORMAT    - stream codec: mp3, opus, aac (default: mp3)
//	STREAM_BITRATE   - stream bitrate (default: 320k)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	libMPV "musicd/lib/player/mpv"
	libQueue "musicd/lib/player/queue"
	libStream "musicd/lib/player/stream"
	"musicd/lib/types"

	"github.com/gorilla/websocket"
)

// ── Types ────────────────────────────────────────────────────────────────────

// Config holds all daemon configuration from environment variables.
type Config struct {
	MusicdURL     string
	SessionName   string
	StateDir      string
	AudioDevice   string
	InitialVolume float64
	MpvSocket     string
	Stream        libStream.Config
}

func loadConfig() Config {
	musicdURL := os.Getenv("MUSICD_URL")
	if musicdURL == "" {
		log.Fatal("MUSICD_URL environment variable not set (e.g. http://localhost:8080)")
	}
	musicdURL = strings.TrimSuffix(musicdURL, "/")

	sessionName := os.Getenv("SESSION_NAME")
	if sessionName == "" {
		hostname, _ := os.Hostname()
		sessionName = hostname + "-player"
	}

	stateDir := os.Getenv("STATE_DIR")
	if stateDir == "" {
		stateDir = "/var/lib/musicplayerd"
	}

	audioDevice := os.Getenv("AUDIO_DEVICE")
	if audioDevice == "" {
		audioDevice = "auto"
	}

	volume := 80.0
	if v := os.Getenv("AUDIO_VOLUME"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			volume = parsed
		}
	}

	mpvSocket := os.Getenv("MPV_SOCKET")
	if mpvSocket == "" {
		mpvSocket = "/tmp/musicplayerd-mpv.sock"
	}

	streamFormat := os.Getenv("STREAM_FORMAT")
	if streamFormat == "" {
		streamFormat = "mp3"
	}
	streamBitrate := os.Getenv("STREAM_BITRATE")
	if streamBitrate == "" {
		streamBitrate = "320k"
	}

	return Config{
		MusicdURL:     musicdURL,
		SessionName:   sessionName,
		StateDir:      stateDir,
		AudioDevice:   audioDevice,
		InitialVolume: volume,
		MpvSocket:     mpvSocket,
		Stream: libStream.Config{
			Enabled:  os.Getenv("STREAM_ENABLED") == "true",
			RTPDest:  os.Getenv("STREAM_RTP_DEST"),
			HTTPPort: os.Getenv("STREAM_HTTP_PORT"),
			Format:   streamFormat,
			Bitrate:  streamBitrate,
		},
	}
}

// ── Daemon ───────────────────────────────────────────────────────────────────

// Daemon is the music player daemon.
type Daemon struct {
	cfg       Config
	mpv       *libMPV.MpvClient
	streams   *libStream.StreamManager
	sessionID string
	q         *libQueue.Queue

	// WebSocket
	wsConn *websocket.Conn
	wsMu   sync.Mutex

	// Playback state (non-queue fields)
	mu              sync.RWMutex
	isPlaying       bool
	currentTime     float64
	duration        float64
	volume          float64 // 0–100
	muted           bool
	currentPlaylist map[string]interface{} // {"id":"...", "name":"..."} or nil
}

func main() {
	cfg := loadConfig()

	// Set up streaming before mpv so we can route mpv to the virtual sink.
	var streams *libStream.StreamManager
	audioDevice := cfg.AudioDevice
	if cfg.Stream.Enabled {
		streams = libStream.NewStreamManager(cfg.Stream)
		if err := streams.Start(); err != nil {
			log.Printf("Warning: streaming failed to start: %v", err)
			streams = nil
		} else {
			// Point mpv at the virtual sink so only its audio is captured.
			audioDevice = streams.SinkName()
			log.Printf("mpv audio routed to virtual sink: %s", audioDevice)
		}
	}

	mpv := libMPV.New(cfg.MpvSocket, audioDevice, cfg.InitialVolume)
	if err := mpv.Start(); err != nil {
		log.Fatalf("Failed to start mpv: %v", err)
	}
	defer mpv.Shutdown()
	if streams != nil {
		defer streams.Stop()
	}

	d := &Daemon{
		cfg:       cfg,
		mpv:       mpv,
		q:         libQueue.NewQueue(),
		streams:   streams,
		sessionID: loadSessionID(cfg.StateDir),
		volume:    cfg.InitialVolume,
	}

	// Wire mpv events.
	mpv.OnTimePos = func(pos float64) {
		d.mu.Lock()
		d.currentTime = pos
		d.mu.Unlock()
	}
	mpv.OnDuration = func(dur float64) {
		d.mu.Lock()
		d.duration = dur
		d.mu.Unlock()
	}
	mpv.OnTrackEnd = func(reason, fileError string) {
		log.Printf("mpv end-file: reason=%s file_error=%s", reason, fileError)
		switch reason {
		case "eof":
			// Natural end of track.
			if d.q.GetRepeatMode() == "One" && d.q.Current() != nil {
				// Repeat One: restart the same track by reloading.
				// (seek doesn't work after EOF, so we reload the file.)
				log.Printf("repeat one — replaying")
				d.playTrack(*d.q.Current())
			} else if track := d.q.Next(); track != nil {
				d.playTrack(*track)
			} else {
				d.mu.Lock()
				d.isPlaying = false
				d.currentTime = 0
				d.duration = 0
				d.mu.Unlock()
			}
			d.broadcastState()
		case "error":
			log.Printf("mpv playback error: %s", fileError)
			d.mu.Lock()
			d.isPlaying = false
			d.mu.Unlock()
			d.broadcastState()
		default:
			// "stop" (loadfile replaced track, or explicit stop) — ignore.
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("musicplayerd starting — server: %s  session: %s", cfg.MusicdURL, cfg.SessionName)
	d.run(ctx)
	log.Printf("musicplayerd stopped")
}

// ── Connection loop ──────────────────────────────────────────────────────────

func (d *Daemon) run(ctx context.Context) {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := d.connect(); err != nil {
			log.Printf("connection error: %v — retrying in %v", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, 30*time.Second)
			continue
		}
		backoff = time.Second

		stopBroadcast := make(chan struct{})
		go d.stateBroadcaster(stopBroadcast)

		if err := d.readLoop(ctx); err != nil {
			log.Printf("websocket error: %v — reconnecting", err)
		}

		close(stopBroadcast)
		d.wsMu.Lock()
		if d.wsConn != nil {
			d.wsConn.Close()
			d.wsConn = nil
		}
		d.wsMu.Unlock()
	}
}

func (d *Daemon) connect() error {
	wsURL := strings.Replace(d.cfg.MusicdURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL += "/api/ws/player"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", wsURL, err)
	}

	hostname, _ := os.Hostname()
	reg := map[string]interface{}{
		"type":            "register",
		"session_id":      d.sessionID,
		"session_name":    d.cfg.SessionName,
		"client_hostname": hostname,
	}
	if err := conn.WriteJSON(reg); err != nil {
		conn.Close()
		return fmt.Errorf("send register: %w", err)
	}

	var ack map[string]string
	if err := conn.ReadJSON(&ack); err != nil {
		conn.Close()
		return fmt.Errorf("read session_ack: %w", err)
	}

	if id := ack["session_id"]; id != "" {
		d.sessionID = id
		saveSessionID(d.cfg.StateDir, id)
	}

	log.Printf("session established: %s (%s)", ack["session_name"], d.sessionID)

	d.wsMu.Lock()
	d.wsConn = conn
	d.wsMu.Unlock()

	d.broadcastState()
	return nil
}

func (d *Daemon) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		d.wsMu.Lock()
		conn := d.wsConn
		d.wsMu.Unlock()

		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}

		msgType, _ := msg["type"].(string)
		switch msgType {
		case "command":
			action, _ := msg["action"].(string)
			go func() {
				d.handleCommand(action, msg["value"])
				d.broadcastState()
			}()
		case "controllers_update":
			count, _ := msg["count"].(float64)
			log.Printf("controllers connected: %d", int(count))
		}
	}
}

// ── Command handling ─────────────────────────────────────────────────────────

func (d *Daemon) handleCommand(action string, value interface{}) {
	log.Printf("command: %s", action)
	switch action {
	case "play":
		d.resume()
		log.Printf("  → play (resumed)")
	case "pause":
		d.pause()
		log.Printf("  → pause")
	case "toggle_play":
		d.mu.RLock()
		playing := d.isPlaying
		d.mu.RUnlock()
		if playing {
			d.pause()
			log.Printf("  → toggle_play → pause")
		} else {
			d.resume()
			log.Printf("  → toggle_play → play")
		}
	case "next":
		log.Printf("  → next track")
		d.next()
	case "previous":
		log.Printf("  → previous track")
		d.previous()
	case "seek":
		if sec, ok := asFloat(value); ok {
			log.Printf("  → seek to %.1fs", sec)
			d.seekTo(sec)
		}
	case "volume":
		if v, ok := asFloat(value); ok {
			log.Printf("  → volume %.1f", v)
			d.setVolume(v)
		} else {
			log.Printf("  → volume: invalid value %T %v", value, value)
		}
	case "toggle_mute":
		d.mu.Lock()
		d.muted = !d.muted
		muted := d.muted
		d.mu.Unlock()
		d.mpv.SetMute(muted)
		log.Printf("  → mute=%v", muted)
	case "set_shuffle":
		if v, ok := value.(bool); ok {
			d.q.SetShuffle(v)
			log.Printf("  → shuffle=%v", v)
		}
	case "set_repeat":
		if v, ok := value.(string); ok {
			d.q.SetRepeatMode(v)
			log.Printf("  → repeat=%s", v)
		}
	case "play_track":
		if m, ok := value.(map[string]interface{}); ok {
			id, _ := m["id"].(string)
			search, _ := m["search"].(string)
			if id != "" {
				d.setCurrentPlaylist(nil)
				d.cmdPlayTrack(id, search)
			}
		}
	case "play_playlist":
		if m, ok := value.(map[string]interface{}); ok {
			if id, ok := m["id"].(string); ok {
				name, _ := m["name"].(string)
				d.setCurrentPlaylist(map[string]interface{}{"id": id, "name": name})
				d.loadAndPlay("/api/playlist/" + id + "/tracks")
			}
		}
	case "play_playlist_track":
		if m, ok := value.(map[string]interface{}); ok {
			id, _ := m["id"].(string)
			playlistPositionID, _ := m["playlist_position_id"].(string)
			pid, _ := m["playlist_id"].(string)
			pname, _ := m["playlist_name"].(string)
			search, _ := m["search"].(string)
			d.setCurrentPlaylist(map[string]interface{}{"id": pid, "name": pname})
			d.loadAndPlayFromPlaylist("/api/playlist/"+pid+"/tracks", id, playlistPositionID, search)
		}
	case "play_album":
		if m, ok := value.(map[string]interface{}); ok {
			if id, ok := m["id"].(string); ok {
				d.setCurrentPlaylist(nil)
				d.loadAndPlay("/api/album/" + id + "/tracks")
			}
		}
	case "play_album_track":
		if m, ok := value.(map[string]interface{}); ok {
			id, _ := m["id"].(string)
			aid, _ := m["album_id"].(string)
			search, _ := m["search"].(string)
			d.setCurrentPlaylist(nil)
			d.loadAndPlayFrom("/api/album/"+aid+"/tracks", id, search)
		}
	case "play_artist":
		if m, ok := value.(map[string]interface{}); ok {
			if id, ok := m["id"].(string); ok {
				d.setCurrentPlaylist(nil)
				d.loadAndPlay("/api/artist/" + id + "/tracks")
			}
		}
	case "play_artist_track":
		if m, ok := value.(map[string]interface{}); ok {
			id, _ := m["id"].(string)
			aid, _ := m["artist_id"].(string)
			search, _ := m["search"].(string)
			d.setCurrentPlaylist(nil)
			d.loadAndPlayFrom("/api/artist/"+aid+"/tracks", id, search)
		}
	case "queue_add":
		if m, ok := value.(map[string]interface{}); ok {
			if id, ok := m["id"].(string); ok {
				d.cmdQueueAdd(id)
			}
		}
	case "queue_clear":
		d.q.Clear()
	}
}

// ── Queue operations ─────────────────────────────────────────────────────────

func (d *Daemon) cmdPlayTrack(id, search string) {
	apiPath := "/api/tracks"
	if search != "" {
		apiPath = "/api/tracks?search=" + url.QueryEscape(search)
	}
	tracks, err := d.fetchTracks(apiPath)
	if err != nil || len(tracks) == 0 {
		track, err := d.fetchTrack(id)
		if err != nil {
			log.Printf("fetchTrack %s: %v", id, err)
			return
		}
		d.q.PlayTracks([]types.Track{track}, 0)
		d.playTrack(track)
		return
	}

	startIdx := 0
	for i, t := range tracks {
		if t.ID == id {
			startIdx = i
			break
		}
	}

	selected := d.q.PlayTracks(tracks, startIdx)
	d.playTrack(selected)
}

func (d *Daemon) cmdQueueAdd(id string) {
	track, err := d.fetchTrack(id)
	if err != nil {
		log.Printf("fetchTrack %s: %v", id, err)
		return
	}
	d.q.Add(track)
}

func (d *Daemon) loadAndPlay(apiPath string) {
	tracks, err := d.fetchTracks(apiPath)
	if err != nil || len(tracks) == 0 {
		log.Printf("fetchTracks %s: %v", apiPath, err)
		return
	}
	// When shuffle is on and no specific track is requested, pick a random start
	// (matches frontend behavior where all tracks are shuffled, then index 0 plays).
	startIdx := 0
	if d.q.IsShuffle() {
		startIdx = rand.Intn(len(tracks))
	}
	selected := d.q.PlayTracks(tracks, startIdx)
	d.playTrack(selected)
}

func (d *Daemon) loadAndPlayFrom(apiPath, trackID, search string) {
	if search != "" {
		apiPath += "?search=" + url.QueryEscape(search)
	}
	tracks, err := d.fetchTracks(apiPath)
	if err != nil || len(tracks) == 0 {
		log.Printf("fetchTracks %s: %v", apiPath, err)
		return
	}
	startIdx := 0
	for i, t := range tracks {
		if t.ID == trackID {
			startIdx = i
			break
		}
	}
	selected := d.q.PlayTracks(tracks, startIdx)
	d.playTrack(selected)
}

// loadAndPlayFromPlaylist is like loadAndPlayFrom but handles duplicate tracks in playlists
// using the playlist_position_id to identify the exact track to play.
func (d *Daemon) loadAndPlayFromPlaylist(apiPath, trackID, playlistPositionID, search string) {
	if search != "" {
		apiPath += "?search=" + url.QueryEscape(search)
	}
	tracks, err := d.fetchTracks(apiPath)
	if err != nil || len(tracks) == 0 {
		log.Printf("fetchTracks %s: %v", apiPath, err)
		return
	}
	startIdx := 0
	// If playlistPositionID is provided, use it (handles duplicates)
	if playlistPositionID != "" {
		for i, t := range tracks {
			if t.PlaylistPositionID == playlistPositionID {
				startIdx = i
				break
			}
		}
	} else {
		// Fall back to track ID if no playlistPositionID
		for i, t := range tracks {
			if t.ID == trackID {
				startIdx = i
				break
			}
		}
	}
	selected := d.q.PlayTracks(tracks, startIdx)
	d.playTrack(selected)
}

// playTrack tells mpv to play a track. Queue state is already set by the caller.
func (d *Daemon) playTrack(track types.Track) {
	d.mu.Lock()
	d.isPlaying = true
	d.currentTime = 0
	d.duration = track.DurationSec
	d.mu.Unlock()

	log.Printf("▶  %s — %s", track.Artist, track.Title)

	audioURL := d.cfg.MusicdURL + "/api/music/" + encodeFilePath(track.FilePath)
	log.Printf("   url: %s", audioURL)
	if err := d.mpv.LoadFile(audioURL); err != nil {
		log.Printf("mpv loadfile: %v", err)
		d.mu.Lock()
		d.isPlaying = false
		d.mu.Unlock()
	}
}

func (d *Daemon) next() {
	if track := d.q.Next(); track != nil {
		d.playTrack(*track)
	} else {
		d.mu.Lock()
		d.isPlaying = false
		d.currentTime = 0
		d.duration = 0
		d.mu.Unlock()
		d.mpv.Stop()
	}
}

func (d *Daemon) previous() {
	if track := d.q.Previous(); track != nil {
		d.playTrack(*track)
	}
}

// ── Playback control ─────────────────────────────────────────────────────────

func (d *Daemon) pause() {
	d.mpv.SetPause(true)
	d.mu.Lock()
	d.isPlaying = false
	d.mu.Unlock()
}

func (d *Daemon) resume() {
	if current := d.q.Current(); current != nil {
		d.mpv.SetPause(false)
		d.mu.Lock()
		d.isPlaying = true
		d.mu.Unlock()
	}
}

func (d *Daemon) seekTo(sec float64) {
	d.mpv.Seek(sec)
}

func (d *Daemon) setVolume(v float64) {
	if v < 0 {
		v = 0
	} else if v > 100 {
		v = 100
	}
	d.mu.Lock()
	d.volume = v
	d.muted = false
	d.mu.Unlock()
	d.mpv.SetVolume(v)
	d.mpv.SetMute(false)
}

// ── Playlist tracking ────────────────────────────────────────────────────────

func (d *Daemon) setCurrentPlaylist(pl map[string]interface{}) {
	d.mu.Lock()
	d.currentPlaylist = pl
	d.mu.Unlock()
}

// ── State broadcast ──────────────────────────────────────────────────────────

func (d *Daemon) broadcastState() {
	qs := d.q.State()
	tq := d.q.TemporaryQueue()
	d.mu.RLock()
	state := map[string]interface{}{
		"type":             "state",
		"is_playing":       d.isPlaying,
		"current_track":    qs.CurrentTrack,
		"current_time":     d.currentTime,
		"duration":         d.duration,
		"volume":           d.volume,
		"muted":            d.muted,
		"shuffle":          qs.Shuffle,
		"repeat_mode":      qs.RepeatMode,
		"queue":            qs.PriorityQueue,
		"temporary_queue":  tq,
		"current_playlist": d.currentPlaylist,
	}
	d.mu.RUnlock()

	d.wsMu.Lock()
	defer d.wsMu.Unlock()
	if d.wsConn != nil {
		if err := d.wsConn.WriteJSON(state); err != nil {
			log.Printf("broadcastState: %v", err)
		}
	}
}

func (d *Daemon) stateBroadcaster(stop chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.broadcastState()
		case <-stop:
			return
		}
	}
}

// ── API helpers ──────────────────────────────────────────────────────────────

func (d *Daemon) fetchTrack(id string) (types.Track, error) {
	resp, err := http.Get(d.cfg.MusicdURL + "/api/track/" + id)
	if err != nil {
		return types.Track{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return types.Track{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var t types.Track
	return t, json.NewDecoder(resp.Body).Decode(&t)
}

// fetchTracks fetches all tracks from a paginated API endpoint.
func (d *Daemon) fetchTracks(apiPath string) ([]types.Track, error) {
	var all []types.Track
	for page := 1; ; page++ {
		sep := "?"
		if strings.Contains(apiPath, "?") {
			sep = "&"
		}
		url := fmt.Sprintf("%s%s%spageSize=500&page=%d", d.cfg.MusicdURL, apiPath, sep, page)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
		}

		// Try flat array first (used by artist/album endpoints).
		var arr []types.Track
		if json.Unmarshal(body, &arr) == nil {
			all = append(all, arr...)
			break
		}

		// Paginated object with "data" field.
		var paged struct {
			Data       []types.Track `json:"data"`
			TotalPages int           `json:"totalPages"`
		}
		if err := json.Unmarshal(body, &paged); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		all = append(all, paged.Data...)
		if page >= paged.TotalPages || len(paged.Data) == 0 {
			break
		}
	}
	return all, nil
}

// ── Session persistence ──────────────────────────────────────────────────────

func sessionFile(stateDir string) string {
	return filepath.Join(stateDir, "musicplayerd-session-id")
}

func loadSessionID(stateDir string) string {
	b, err := os.ReadFile(sessionFile(stateDir))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func saveSessionID(stateDir, id string) {
	os.MkdirAll(stateDir, 0755)
	if err := os.WriteFile(sessionFile(stateDir), []byte(id), 0600); err != nil {
		log.Printf("saveSessionID: %v", err)
	}
}

// ── Utilities ────────────────────────────────────────────────────────────────

// encodeFilePath percent-encodes each segment of a file path for use in a URL.
func encodeFilePath(filePath string) string {
	parts := strings.Split(filePath, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func asFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}
