package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/abema/go-mp4"
	"github.com/dhowden/tag"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"golang.org/x/text/unicode/norm"
)

//go:embed all:static
var _staticFS embed.FS
var staticFS, _ = fs.Sub(_staticFS, "static")

//go:embed all:ui
var _uiFS embed.FS
var uiFS, _ = fs.Sub(_uiFS, "ui")

var db *sql.DB
var adminUserID string

// Scan state for concurrency control
var scanMutex sync.Mutex
var isScanning int32 // atomic: 0=idle, 1=scanning

// WebSocket upgrader for remote control sessions
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // allow all origins for Wireguard
}

// Remote control session management
type PlayerSession struct {
	ID             string
	Name           string
	PlayerConn     *websocket.Conn
	Controllers    []*websocket.Conn
	LastState      json.RawMessage
	DisconnectedAt *time.Time
	mu             sync.Mutex
	playerWriteMu  sync.Mutex // serialises writes to PlayerConn
	controllerMus  map[*websocket.Conn]*sync.Mutex
}

// writeToPlayer serialises a write to the player connection.
// Caller must NOT hold session.mu while calling this.
func (s *PlayerSession) writeToPlayer(msg interface{}) error {
	s.mu.Lock()
	pc := s.PlayerConn
	s.mu.Unlock()
	if pc == nil {
		return fmt.Errorf("no player connection")
	}
	s.playerWriteMu.Lock()
	defer s.playerWriteMu.Unlock()
	return pc.WriteJSON(msg)
}

// writeToController serialises a write to a specific controller connection.
func (s *PlayerSession) writeToController(cc *websocket.Conn, msg interface{}) error {
	s.mu.Lock()
	mu, ok := s.controllerMus[cc]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("controller not found")
	}
	mu.Lock()
	defer mu.Unlock()
	return cc.WriteJSON(msg)
}

// broadcastToControllers sends a message to all controllers (non-blocking per connection).
func (s *PlayerSession) broadcastToControllers(msg interface{}) {
	s.mu.Lock()
	controllers := make([]*websocket.Conn, len(s.Controllers))
	copy(controllers, s.Controllers)
	s.mu.Unlock()

	for _, cc := range controllers {
		s.writeToController(cc, msg)
	}
}

type SessionHub struct {
	sessions map[string]*PlayerSession
	mu       sync.RWMutex
}

var hub = &SessionHub{sessions: make(map[string]*PlayerSession)}
var serverHostname string

type Track struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Year        int    `json:"year"`
	FilePath    string `json:"file_path"`
	FileHash    string `json:"file_hash"`
	CoverArtID  string `json:"cover_art_id"`
	Duration    string `json:"duration"`
	DurationSec int    `json:"duration_seconds"`
}

type Playlist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	TrackCount  int    `json:"track_count"`
	CoverArtID  string `json:"cover_art_id"`
}

type PlaylistData struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Path        string  `json:"path"`
	Tracks      []Track `json:"tracks"`
	CoverArtID  string  `json:"cover_art_id"`
}

type Album struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Artist     string  `json:"artist"`
	Year       int     `json:"year"`
	Tracks     []Track `json:"tracks"`
	TrackCount int     `json:"track_count"`
	CoverArtID string  `json:"cover_art_id"`
}

type Artist struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	CoverArtID string   `json:"cover_art_id"`
	Albums     []*Album `json:"albums"`
	Tracks     []Track  `json:"tracks"` // All tracks by this artist
	TrackCount int      `json:"track_count"`
}

// MusicLibrary struct removed - now using database-only approach

type CreatePlaylistRequest struct {
	Name       string `json:"name"`
	Location   string `json:"location"`   // "music", "playlists", or "custom"
	CustomPath string `json:"customPath"` // only used if location is "custom"
}

type PageData struct {
	Tracks     []Track
	Playlists  []Playlist
	Page       int
	PageSize   int
	TotalPages int
	Search     string
}

type PlaylistPageData struct {
	Playlist          Playlist
	PaginatedPlaylist Playlist
	AllPlaylists      []Playlist
	Page              int
	PageSize          int
	TotalPages        int
	Search            string
}

type ArtistsPageData struct {
	Artists    []Artist
	Playlists  []Playlist
	Page       int
	PageSize   int
	TotalPages int
	Search     string
}

type ArtistPageData struct {
	Artist          Artist
	Albums          []Album
	PaginatedTracks []Track
	AllPlaylists    []Playlist
	Page            int
	PageSize        int
	TotalPages      int
	Search          string
}

type AlbumPageData struct {
	Album        Album
	Artist       string
	AllPlaylists []Playlist
	Page         int
	PageSize     int
	TotalPages   int
	Search       string
}

// Removed global library variable - using database-only approach
var musicDir = "./music"         // Configure this path
var playlistsDir = "./playlists" // Configure this path for new playlists
var pathPrefix = ""              // Configure this path prefix for reverse proxy setups

// Helper function to add path prefix to routes
func prefixPath(path string) string {
	if pathPrefix == "" {
		return path
	}
	return pathPrefix + path
}

func musicFilePath(filePath string) string {
	return strings.TrimPrefix(filePath, musicDir+"/")
}

func createCoverArt(imageData []byte, mimeType string) (string, error) {
	id := uuid.New().String()
	query := `
		INSERT INTO images (id, content, mime_type, uploaded_by, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := db.Exec(query, id, imageData, mimeType, adminUserID)
	if err != nil {
		return "", err
	}

	return id, nil
}

func findExistingCoverArt(imageData []byte) (string, error) {
	// Generate a hash of the image data for comparison
	// We'll use a simple approach by comparing the actual binary data
	query := `SELECT id FROM images WHERE content = $1 LIMIT 1`

	var existingID string
	err := db.QueryRow(query, imageData).Scan(&existingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No existing cover art found
		}
		return "", err
	}

	return existingID, nil
}

func createOrFindCoverArt(imageData []byte, mimeType string) (string, error) {
	// First check if this cover art already exists
	existingID, err := findExistingCoverArt(imageData)
	if err != nil {
		return "", err
	}

	if existingID != "" {
		// log.Printf("Using existing cover art with ID: %s", existingID)
		return existingID, nil
	}

	// Create new cover art if it doesn't exist
	log.Printf("Creating new cover art")
	return createCoverArt(imageData, mimeType)
}

func createOrFindArtist(name string, coverArtID string) (string, error) {
	if name == "" {
		name = "Unknown Artist"
	}

	// Try to find existing artist
	var artistID string
	err := db.QueryRow("SELECT id FROM artists WHERE name = $1", name).Scan(&artistID)
	if err == nil {
		// Update cover art if provided and artist doesn't have one
		if coverArtID != "" {
			var existingCoverArtID *string
			err = db.QueryRow("SELECT cover_art_id FROM artists WHERE id = $1", artistID).Scan(&existingCoverArtID)
			if err == nil && (existingCoverArtID == nil || *existingCoverArtID == "") {
				_, err = db.Exec("UPDATE artists SET cover_art_id = $1, updated_at = NOW() WHERE id = $2", coverArtID, artistID)
				if err != nil {
					log.Printf("Error updating artist cover art: %v", err)
				}
			}
		}
		return artistID, nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// Create new artist - let PostgreSQL generate the UUID
	var coverArtIDPtr *string
	if coverArtID != "" {
		coverArtIDPtr = &coverArtID
	}

	err = db.QueryRow(`
		INSERT INTO artists (name, cover_art_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id
	`, name, coverArtIDPtr).Scan(&artistID)

	return artistID, err
}

func createOrFindAlbum(title string, artistID string, year int, coverArtID string) (string, error) {
	if title == "" {
		title = "Unknown Album"
	}

	// Try to find existing album for this artist
	var albumID string
	err := db.QueryRow("SELECT id FROM albums WHERE title = $1 AND artist_id = $2", title, artistID).Scan(&albumID)
	if err == nil {
		// Update year and cover art if provided
		updateNeeded := false
		updateQuery := "UPDATE albums SET updated_at = NOW()"
		args := []interface{}{}
		argIndex := 1

		if year > 0 {
			var existingYear *int
			err = db.QueryRow("SELECT year FROM albums WHERE id = $1", albumID).Scan(&existingYear)
			if err == nil && (existingYear == nil || *existingYear == 0) {
				updateQuery += fmt.Sprintf(", year = $%d", argIndex)
				args = append(args, year)
				argIndex++
				updateNeeded = true
			}
		}

		if coverArtID != "" {
			var existingCoverArtID *string
			err = db.QueryRow("SELECT cover_art_id FROM albums WHERE id = $1", albumID).Scan(&existingCoverArtID)
			if err == nil && (existingCoverArtID == nil || *existingCoverArtID == "") {
				updateQuery += fmt.Sprintf(", cover_art_id = $%d", argIndex)
				args = append(args, coverArtID)
				argIndex++
				updateNeeded = true
			}
		}

		if updateNeeded {
			updateQuery += fmt.Sprintf(" WHERE id = $%d", argIndex)
			args = append(args, albumID)
			_, err = db.Exec(updateQuery, args...)
			if err != nil {
				log.Printf("Error updating album: %v", err)
			}
		}

		return albumID, nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// Create new album - let PostgreSQL generate the UUID
	var yearPtr *int
	if year > 0 {
		yearPtr = &year
	}
	var coverArtIDPtr *string
	if coverArtID != "" {
		coverArtIDPtr = &coverArtID
	}

	err = db.QueryRow(`
		INSERT INTO albums (title, artist_id, year, cover_art_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`, title, artistID, yearPtr, coverArtIDPtr).Scan(&albumID)

	return albumID, err
}

func trackNeedsUpdate(existingTrack, newTrack Track) bool {
	// First check file hash - if it's the same, the file hasn't changed at all
	// This is much more efficient than comparing metadata fields
	if existingTrack.FileHash == "" {
		log.Printf("WARNING: No existing file hash for track %s, your database is out of date", existingTrack.FilePath)
	}

	if existingTrack.FileHash != "" && newTrack.FileHash != "" {
		if existingTrack.FileHash == newTrack.FileHash {
			// File hasn't changed, no update needed
			return false
		}
		// File has changed, update needed
		log.Printf("File hash changed for track %s: %s -> %s", existingTrack.FilePath, existingTrack.FileHash[:8], newTrack.FileHash[:8])
		return true
	}

	// Fallback to metadata comparison if hash is not available
	// This handles cases where we're migrating from old schema without hashes
	if existingTrack.Title != newTrack.Title ||
		existingTrack.Artist != newTrack.Artist ||
		existingTrack.Album != newTrack.Album ||
		existingTrack.Year != newTrack.Year ||
		existingTrack.DurationSec != newTrack.DurationSec {
		return true
	}

	// Check if cover art has changed
	if existingTrack.CoverArtID != newTrack.CoverArtID {
		return true
	}

	return false
}

func insertNewSong(track Track) error {
	// Create or find the artist first
	_, err := createOrFindArtist(track.Artist, track.CoverArtID)
	if err != nil {
		return fmt.Errorf("error creating/finding artist: %v", err)
	}

	// Create or find the album (this will also create the artist if needed)
	artistID, err := createOrFindArtist(track.Artist, track.CoverArtID)
	if err != nil {
		return fmt.Errorf("error creating/finding artist for album: %v", err)
	}

	_, err = createOrFindAlbum(track.Album, artistID, track.Year, track.CoverArtID)
	if err != nil {
		return fmt.Errorf("error creating/finding album: %v", err)
	}

	query := `
		INSERT INTO songs (id, title, artist, album, year, file_path, file_hash, cover_art_id, duration, uploaded_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`

	var coverArtID *string
	if track.CoverArtID != "" {
		coverArtID = &track.CoverArtID
	}

	_, err = db.Exec(query, track.ID, track.Title, track.Artist, track.Album, track.Year, track.FilePath, track.FileHash, coverArtID, track.DurationSec, adminUserID)
	return err
}

func updateExistingSong(track Track) error {
	// Create or find the artist
	artistID, err := createOrFindArtist(track.Artist, track.CoverArtID)
	if err != nil {
		return fmt.Errorf("error creating/finding artist: %v", err)
	}

	// Create or find the album
	_, err = createOrFindAlbum(track.Album, artistID, track.Year, track.CoverArtID)
	if err != nil {
		return fmt.Errorf("error creating/finding album: %v", err)
	}

	query := `
		UPDATE songs SET
			title = $2,
			artist = $3,
			album = $4,
			year = $5,
			file_hash = $6,
			cover_art_id = $7,
			duration = $8,
			updated_at = NOW()
		WHERE file_path = $1
	`

	var coverArtID *string
	if track.CoverArtID != "" {
		coverArtID = &track.CoverArtID
	}

	_, err = db.Exec(query, track.FilePath, track.Title, track.Artist, track.Album, track.Year, track.FileHash, coverArtID, track.DurationSec)
	return err
}

func getExistingSongs() (map[string]Track, error) {
	query := `
		SELECT id, title, artist, album, year, file_path, file_hash, cover_art_id, duration
		FROM songs
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	songs := make(map[string]Track)
	for rows.Next() {
		var track Track
		var coverArtID sql.NullString
		var fileHash sql.NullString
		err := rows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &track.Year, &track.FilePath, &fileHash, &coverArtID, &track.DurationSec)
		if err != nil {
			return nil, err
		}

		if coverArtID.Valid {
			track.CoverArtID = coverArtID.String
		}
		if fileHash.Valid {
			track.FileHash = fileHash.String
		}

		songs[track.FilePath] = track
	}

	return songs, nil
}

func deleteRemovedSongs(existingPaths []string) error {
	if len(existingPaths) == 0 {
		return nil
	}

	placeholders := make([]string, len(existingPaths))
	args := make([]interface{}, len(existingPaths))

	for i, path := range existingPaths {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = path
	}

	query := fmt.Sprintf("DELETE FROM songs WHERE file_path NOT IN (%s)", strings.Join(placeholders, ","))
	_, err := db.Exec(query, args...)
	return err
}

func getCoverArt(coverArtID string) ([]byte, string, error) {
	query := `SELECT content, mime_type FROM images WHERE id = $1`

	var content []byte
	var mimeType string
	err := db.QueryRow(query, coverArtID).Scan(&content, &mimeType)
	if err != nil {
		return nil, "", err
	}

	return content, mimeType, nil
}

func getAdminUserID() (string, error) {
	var userID string
	query := `SELECT id FROM users WHERE username = 'admin' LIMIT 1`
	err := db.QueryRow(query).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("failed to get admin user ID: %v", err)
	}
	return userID, nil
}

// loadTracksFromDatabase function removed - using database queries directly in handlers

func main() {
	musicDir = os.Getenv("MUSIC_DIR")
	if musicDir == "" {
		log.Fatal("MUSIC_DIR environment variable not set")
	}
	musicDir = filepath.Clean(musicDir)

	playlistsDir = os.Getenv("PLAYLISTS_DIR")
	if playlistsDir == "" {
		log.Fatal("PLAYLISTS_DIR environment variable not set")
	}
	playlistsDir = filepath.Clean(playlistsDir)

	pathPrefix = os.Getenv("PATH_PREFIX")
	if pathPrefix != "" {
		pathPrefix = strings.TrimSuffix(pathPrefix, "/")
		if !strings.HasPrefix(pathPrefix, "/") {
			pathPrefix = "/" + pathPrefix
		}
	}

	// Get server hostname for remote session names
	var err error
	serverHostname, err = os.Hostname()
	if err != nil {
		log.Println("Warning: could not get hostname, using 'musicd'")
		serverHostname = "musicd"
	}

	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}
	log.Println("Successfully connected to database")

	// Get admin user ID for song insertions
	adminUserID, err = getAdminUserID()
	if err != nil {
		log.Fatalf("Error getting admin user ID: %v", err)
	}
	log.Printf("Using admin user ID: %s", adminUserID)

	log.Printf("Using music directory: %s", musicDir)
	log.Printf("Using playlists directory: %s", playlistsDir)
	log.Printf("Using path prefix: %s", pathPrefix)

	// Check if Music dir exists
	if _, err := os.Stat(musicDir); os.IsNotExist(err) {
		log.Fatal("Music directory does not exist")
	}

	// Check if Playlists dir exists
	if _, err := os.Stat(playlistsDir); os.IsNotExist(err) {
		log.Fatal("Playlists directory does not exist")
	}

	// Scan music library on startup (non-blocking so the server starts immediately)
	atomic.StoreInt32(&isScanning, 1)
	go func() {
		defer atomic.StoreInt32(&isScanning, 0)
		log.Println("Starting initial library scan...")
		if err := scanMusicLibrary(); err != nil {
			log.Printf("Error scanning music library: %v", err)
		}
	}()

	// Start file system watcher for automatic rescan
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("Error creating file watcher: %v", err)
			return
		}
		defer watcher.Close()

		// Recursively add all directories to the watcher
		addWatchDirs := func(rootDir string) error {
			return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					if err := watcher.Add(path); err != nil {
						log.Printf("Error watching %s: %v", path, err)
					}
				}
				return nil
			})
		}

		// Add initial directories
		if err := addWatchDirs(musicDir); err != nil {
			log.Printf("Error adding music directory to watcher: %v", err)
		}
		if rel, err := filepath.Rel(musicDir, playlistsDir); err != nil || strings.HasPrefix(rel, "..") {
			if err := addWatchDirs(playlistsDir); err != nil {
				log.Printf("Error adding playlists directory to watcher: %v", err)
			}
		}

		log.Println("File system watcher started")

		// Debounce FS events - wait 30 seconds after last event before rescanning
		var debounceTimer *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Log the event
				log.Printf("File system event: %s %s", event.Op, event.Name)

				// For directory creation, add to watcher
				if event.Op&fsnotify.Create == fsnotify.Create {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						if err := watcher.Add(event.Name); err != nil {
							log.Printf("Error watching new directory %s: %v", event.Name, err)
						}
					}
				}

				// Debounce: reset timer on each event
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(30*time.Second, func() {
					if atomic.CompareAndSwapInt32(&isScanning, 0, 1) {
						go func() {
							defer atomic.StoreInt32(&isScanning, 0)
							log.Println("Auto-rescanning library due to file system changes...")
							if err := scanMusicLibrary(); err != nil {
								log.Printf("Error during auto-rescan: %v", err)
							}
						}()
					} else {
						log.Println("Skipping auto-rescan: scan already in progress")
					}
				})

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	r := mux.NewRouter()

	// Static files
	staticPath := prefixPath("/static/")
	r.PathPrefix(staticPath).Handler(http.StripPrefix(staticPath, http.FileServer(http.FS(staticFS))))

	// Nuxt PWA files - handled by SPA fallback below
	// uiPath := prefixPath("/ui/")
	// r.PathPrefix(uiPath).Handler(http.StripPrefix(uiPath, http.FileServer(http.FS(uiFS))))

	// Serve music files
	musicPath := prefixPath("/music/")
	r.PathPrefix(musicPath).Handler(http.StripPrefix(musicPath, http.FileServer(http.Dir(musicDir))))

	// Redirect / to /ui/index.html
	r.HandleFunc(prefixPath("/"), func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, prefixPath("/ui/"), http.StatusFound)
	})

	// Serve favicon
	r.HandleFunc(prefixPath("/favicon.ico"), func(w http.ResponseWriter, r *http.Request) {
		// Serve the favicon.ico file from the static fs directly
		faviconFile, err := uiFS.Open("favicon.ico")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer faviconFile.Close()
		w.Header().Set("Content-Type", "image/x-icon")
		io.Copy(w, faviconFile)
	})

	// Serve the Vue.js frontend
	// r.HandleFunc(prefixPath("/"), frontendHandler).Methods("GET")
	// r.PathPrefix(prefixPath("/")).Handler(http.StripPrefix(pathPrefix, http.FileServer(http.Dir("./web/music/frontend/dist")))).Methods("GET")

	// JSON API Routes
	api := r.PathPrefix(prefixPath("/api")).Subrouter()
	api.PathPrefix(musicPath).Handler(http.StripPrefix(prefixPath("/api/music/"), http.FileServer(http.Dir(musicDir))))
	api.HandleFunc("/tracks", apiTracksHandler).Methods("GET")
	api.HandleFunc("/track/{id}", apiTrackByIDHandler).Methods("GET")
	api.HandleFunc("/search", apiSearchHandler).Methods("GET")
	api.HandleFunc("/playlists", apiPlaylistsHandler).Methods("GET")
	api.HandleFunc("/playlist/create", apiCreatePlaylistHandler).Methods("POST")
	// New ID-based playlist routes
	api.HandleFunc("/playlist/{id}", apiPlaylistByIDHandler).Methods("GET")
	api.HandleFunc("/playlist/{id}/tracks", apiPlaylistTracksByIDHandler).Methods("GET")
	api.HandleFunc("/playlist/{id}/add/{trackId}", apiAddToPlaylistByIDHandler).Methods("POST")
	api.HandleFunc("/playlist/{id}/remove/{trackId}", apiRemoveFromPlaylistByIDHandler).Methods("DELETE")
	api.HandleFunc("/playlist/{id}", apiDeletePlaylistHandler).Methods("DELETE")
	api.HandleFunc("/scan-status", apiScanStatusHandler).Methods("GET")
	api.HandleFunc("/rescan", apiRescanHandler).Methods("POST")
	api.HandleFunc("/artists", apiArtistsHandler).Methods("GET")
	api.HandleFunc("/artist/{id}", apiArtistByIDHandler).Methods("GET")
	api.HandleFunc("/artist/{id}/tracks", apiArtistTracksByIDHandler).Methods("GET")
	api.HandleFunc("/album/{id}", apiAlbumByIDHandler).Methods("GET")
	api.HandleFunc("/album/{id}/tracks", apiAlbumTracksByIDHandler).Methods("GET")
	api.HandleFunc("/cover-art/{id}", apiCoverArtHandler).Methods("GET")
	api.HandleFunc("/lyrics", apiLyricsHandler).Methods("GET")

	// Remote control WebSocket endpoints
	api.HandleFunc("/sessions", apiSessionsHandler).Methods("GET")
	api.HandleFunc("/ws/player", apiPlayerWSHandler)
	api.HandleFunc("/ws/control/{id}", apiControlWSHandler)

	// SPA fallback: serve 200.html for any unmatched UI routes
	r.PathPrefix(prefixPath("/ui/")).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the requested file first
		filePath := strings.TrimPrefix(r.URL.Path, prefixPath("/ui/"))
		if filePath == "" {
			filePath = "index.html"
		}

		file, err := uiFS.Open(filePath)
		if err == nil {
			defer file.Close()
			// File exists, read and serve it
			content, err := io.ReadAll(file)
			if err == nil {
				contentType := mime.TypeByExtension(filepath.Ext(filePath))
				if contentType == "" {
					contentType = "application/octet-stream"
				}
				w.Header().Set("Content-Type", contentType)
				w.Header().Set("Cache-Control", "public, max-age=3600")
				w.Write(content)
				return
			}
		}

		// File doesn't exist, serve 200.html for client-side routing
		fallbackFile, err := uiFS.Open("200.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer fallbackFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, fallbackFile)
	}).Methods("GET")

	fmt.Println("Music player starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// buildArtistsAndAlbums organizes tracks into artists and albums
// buildArtistsAndAlbums function removed - using database queries directly in handlers

// These patterns try to catch the binary "data" chunks that are
// surrounded by control / NUL bytes, without killing legitimate
// uses of the word "data" in normal text.
var (
	id3DataPrefix = regexp.MustCompile(`[\x00-\x1F\x7F]+data`)
	id3DataSuffix = regexp.MustCompile(`data[\x00-\x1F\x7F]+`)
)

// CleanID3Text normalizes ID3-derived text, removing binary "data"
// blobs, NULs, and control characters, and ensuring valid UTF-8.
func cleanString(s string) string {
	if s == "" {
		return ""
	}

	// 0. Remove the "data" chunks that are clearly part of binary blobs.
	//    We do this *before* generic cleaning so the word "data" doesn't survive.
	s = id3DataPrefix.ReplaceAllString(s, " ")
	s = id3DataSuffix.ReplaceAllString(s, " ")

	// 1. Remove NUL bytes outright.
	s = strings.ReplaceAll(s, "\x00", "")

	// 2. Ensure valid UTF-8; if not, best-effort decode.
	if !utf8.ValidString(s) {
		var buf bytes.Buffer
		for len(s) > 0 {
			r, size := utf8.DecodeRuneInString(s)
			if r == utf8.RuneError && size == 1 {
				// Skip invalid byte
				s = s[size:]
				continue
			}
			buf.WriteRune(r)
			s = s[size:]
		}
		s = buf.String()
	}

	// 3. Strip control runes (keep newline/tab if you want them).
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 32 || r == 127 || unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)

	// 4. Normalize Unicode.
	s = norm.NFC.String(s)

	// 5. Collapse repeated whitespace and trim.
	s = strings.Join(strings.Fields(s), " ")

	return s
}

func scanMusicLibrary() error {
	startTime := time.Now()
	log.Println("Starting library scan...")

	// Get existing songs from database
	existingSongs, err := getExistingSongs()
	if err != nil {
		log.Printf("Error getting existing songs: %v", err)
		existingSongs = make(map[string]Track)
	}

	currentPaths := []string{}
	m3uFiles := []string{}

	err = filepath.Walk(musicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".mp3" || ext == ".ogg" || ext == ".m4a" || ext == ".flac" {
			filePath := musicFilePath(path)
			currentPaths = append(currentPaths, filePath)

			// Compute file hash for quick comparison
			fileHash, err := computeFileHash(path)
			if err != nil {
				log.Printf("Error computing hash for %s: %v", path, err)
				// Continue with metadata extraction as fallback
			}

			// Check if this file already exists in database
			if existingTrack, exists := existingSongs[filePath]; exists {
				if existingTrack.FileHash == "" {
					log.Printf("WARNING: No file hash for existing track %s, your database is out of date", filePath)
				}

				if existingTrack.FileHash == fileHash {
					return nil // No changes, skip
				}

				// File has changed or no hash available, extract metadata
				track, err := extractMetadata(path, existingTrack.ID)
				if err != nil {
					log.Printf("Error extracting metadata from %s: %v", path, err)
					return nil
				}

				track.FileHash = fileHash

				// Only update if actually needed (final check with all metadata)
				if trackNeedsUpdate(existingTrack, track) {
					log.Printf("Updating song: %s", filePath)
					if err := updateExistingSong(track); err != nil {
						log.Printf("Error updating song %s: %v", filePath, err)
					}
				}
			} else {
				// New file, extract metadata
				track, err := extractMetadata(path, "")
				if err != nil {
					log.Printf("Error extracting metadata from %s: %v", path, err)
					return nil
				}

				track.FileHash = fileHash

				log.Printf("Adding new song: %s", filePath)
				if err := insertNewSong(track); err != nil {
					log.Printf("Error inserting song %s: %v", filePath, err)

					// Log track as json for debugging
					trackJSON, _ := json.Marshal(track)
					log.Printf("Track data: %s", trackJSON)

				}
			}

		} else if ext == ".m3u" {
			m3uFiles = append(m3uFiles, path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Remove songs that no longer exist on disk
	if err := deleteRemovedSongs(currentPaths); err != nil {
		log.Printf("Error removing deleted songs: %v", err)
	}

	// Also scan the playlists directory for .m3u files
	// But first ensure that the playlistsDir is not a child of musicDir to avoid double scanning
	rel, err := filepath.Rel(musicDir, playlistsDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		// Only scan if playlistsDir is not inside musicDir
		err = filepath.Walk(playlistsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".m3u" {
				m3uFiles = append(m3uFiles, path)
			}

			return nil
		})

		if err != nil {
			return err
		}
	} else {
		log.Printf("Playlists directory %s is inside music directory %s; skipping duplicate scan", playlistsDir, musicDir)
	}

	// Load playlists into database
	if len(m3uFiles) > 0 {
		if err := loadPlaylistsIntoDatabase(m3uFiles); err != nil {
			log.Printf("Error loading playlists into database: %v", err)
		}
	}

	// Database-only approach - removed in-memory library operations
	elapsed := time.Since(startTime)
	log.Printf("Music library scan completed - using database-only approach in %s", elapsed)

	// Playlists are loaded dynamically when needed

	// Database-only scan completed
	return nil
}

// extractM4ACoverArt returns the first embedded cover image bytes and a MIME type.
func extractM4ACoverArt(file *os.File) ([]byte, string, error) {
	// Rewind
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, "", err
	}

	var img []byte
	ext := "application/octet-stream"

	_, err := mp4.ReadBoxStructure(file, func(h *mp4.ReadHandle) (interface{}, error) {
		// Read the payload for supported boxes
		if h.BoxInfo.IsSupportedType() {
			box, _, err := h.ReadPayload()
			if err != nil {
				return nil, err
			}

			// Is this a 'data' box under a path that includes 'covr'?
			if h.BoxInfo.Type == mp4.BoxTypeData() {
				foundCovr := false
				for _, anc := range h.Path {
					if anc == mp4.StrToBoxType("covr") {
						foundCovr = true
						break
					}
				}
				if foundCovr {
					if data, ok := box.(*mp4.Data); ok && len(data.Data) > 0 {
						switch data.DataType { // Apple MP4 cover types
						case 13:
							ext = "image/jpeg"
						case 14:
							ext = "image/png"
						case 27:
							ext = "image/bmp"
						default:
							ext = detectImageMIMEType(data.Data)
						}
						img = append([]byte(nil), data.Data...)
						// Stop traversal once we got the first cover image
						return nil, io.EOF
					}
				}
			}

			// Keep descending
			return h.Expand()
		}
		// If the type isn't recognized, just skip it (no descend)
		return nil, nil
	})
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, "", err
	}
	if len(img) == 0 {
		return nil, "", fmt.Errorf("no embedded cover art found")
	}
	return img, ext, nil
}

func detectImageMIMEType(data []byte) string {
	if len(data) < 8 {
		return "image/jpeg" // default fallback
	}

	// Check for JPEG
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}

	// Check for PNG
	if len(data) >= 8 && string(data[1:4]) == "PNG" {
		return "image/png"
	}

	// Check for GIF
	if len(data) >= 6 && string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a" {
		return "image/gif"
	}

	// Check for WebP
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	// Default to JPEG
	return "image/jpeg"
}

// computeFileHash computes the SHA-256 hash of a file
func computeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func extractMetadata(filePath string, trackID string) (Track, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Track{}, err
	}
	defer file.Close()

	m, err := tag.ReadFrom(file)
	if err != nil {
		return Track{}, err
	}

	// Generate UUID if not provided
	if trackID == "" {
		trackID = uuid.New().String()
	}

	track := Track{
		ID:       trackID,
		Filename: filepath.Base(filePath),
		Title:    cleanString(m.Title()),
		Artist:   cleanString(m.Artist()),
		Album:    cleanString(m.Album()),
		Year:     m.Year(),
		FilePath: musicFilePath(filePath),
	}

	// Extract and store cover art separately
	var coverArtData []byte
	var mimeType string

	// Extract cover art if present
	if picture := m.Picture(); picture != nil {
		coverArtData = picture.Data
		mimeType = picture.MIMEType
	} else {
		// M4A files may have cover art that the tag library can't read
		// Attempt to read it manually using MP4 parser
		if picData, picType, err := extractM4ACoverArt(file); err == nil {
			coverArtData = picData
			mimeType = picType
		}
	}

	// Store cover art in database if we found any
	if len(coverArtData) > 0 {
		coverArtID, err := createOrFindCoverArt(coverArtData, mimeType)
		if err != nil {
			log.Printf("Error storing cover art for %s: %v", filePath, err)
		} else {
			track.CoverArtID = coverArtID
		}
	}

	// Extract duration in seconds if available
	// Note: the tag library doesn't provide direct access to duration
	// We'll set duration to 0 for now, could be enhanced later with additional libraries

	// Use filename as title if no title in metadata
	if track.Title == "" {
		track.Title = strings.TrimSuffix(track.Filename, filepath.Ext(track.Filename))
	}

	if track.Artist == "" {
		log.Printf("Warning: No artist metadata for file %s", filePath)

		// Try to split the filename for common "Artist - Title" pattern
		parts := strings.SplitN(track.Title, " - ", 2)
		if len(parts) == 2 {
			track.Artist = cleanString(strings.TrimSpace(parts[0]))

			if track.Title == "" {
				track.Title = cleanString(strings.TrimSpace(parts[1]))
			}
		}
	}

	if track.Title == "" {
		log.Printf("Warning: No title metadata for file %s", filePath)

		parts := strings.SplitN(track.Title, " - ", 2)
		if len(parts) == 2 {
			if track.Title == "" {
				track.Title = cleanString(strings.TrimSpace(parts[1]))
			}
		} else if len(parts) == 1 {
			track.Title = cleanString(strings.TrimSpace(parts[0]))
		}
	}

	if track.Album == "" {
		log.Printf("Warning: No album metadata for file %s", filePath)
	}

	return track, nil
}

// Playlist parsing and database loading functions
func parseM3UFile(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read m3u file: %v", err)
	}

	var tracks []string
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, comments, and #EXTM3U headers
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle relative paths by making them absolute relative to the m3u file
		if !filepath.IsAbs(line) {
			line = filepath.Join(filepath.Dir(filePath), line)
		}

		tracks = append(tracks, musicFilePath(line))
	}

	return tracks, nil
}

func createOrFindPlaylist(name, filePath string) (string, error) {
	var playlistID string

	// First try to find existing playlist by file_path
	err := db.QueryRow("SELECT id FROM playlists WHERE file_path = $1", filePath).Scan(&playlistID)
	if err == nil {
		return playlistID, nil
	}

	// If not found by path, try to find by name
	err = db.QueryRow("SELECT id FROM playlists WHERE name = $1", name).Scan(&playlistID)
	if err == nil {
		// Update the existing playlist with the file_path
		_, err = db.Exec("UPDATE playlists SET file_path = $1, updated_at = NOW() WHERE id = $2", filePath, playlistID)
		return playlistID, err
	}

	// If not found, create new playlist
	err = db.QueryRow(`
		INSERT INTO playlists (name, file_path, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`, name, filePath, adminUserID).Scan(&playlistID)

	return playlistID, err
}

func loadPlaylistTracks(playlistID string, trackPaths []string) error {
	// Clear existing tracks
	_, err := db.Exec("DELETE FROM playlist_songs WHERE playlist_id = $1", playlistID)
	if err != nil {
		return fmt.Errorf("failed to clear existing playlist tracks: %v", err)
	}

	// Add new tracks
	coverArtSet := false

	for position, trackPath := range trackPaths {
		var songID, coverArtID string

		if !coverArtSet {
			// Find song by file path and it's cover art id
			err := db.QueryRow("SELECT id, cover_art_id FROM songs WHERE file_path = $1", trackPath).Scan(&songID, &coverArtID)
			if err == nil {
				// Set our playlist cover art if we haven't done so yet
				if coverArtID != "" {
					_, err := db.Exec("UPDATE playlists SET cover_art_id = $1 WHERE id = $2", coverArtID, playlistID)
					if err != nil {
						log.Printf("Warning: Failed to set cover art for playlist %s: %v", playlistID, err)
					} else {
						coverArtSet = true
					}
				}
			}
		}

		if songID == "" {
			// Find song by file path and it's cover art id
			err := db.QueryRow("SELECT id FROM songs WHERE file_path = $1", trackPath).Scan(&songID)
			if err != nil {
				log.Printf("Warning: Track not found in database for path %s in playlist %s", trackPath, playlistID)
				continue
			}
		}

		// Insert track into playlist
		_, err = db.Exec(`
			INSERT INTO playlist_songs (playlist_id, song_id, position)
			VALUES ($1, $2, $3)
		`, playlistID, songID, position+1)

		if err != nil {
			// Ignore duplicate entries
			if !strings.Contains(err.Error(), "duplicate key value") {
				log.Printf("Warning: Failed to add track %s to playlist %s: %v", trackPath, playlistID, err)
			}
		}
	}

	return nil
}

func loadPlaylistsIntoDatabase(m3uFiles []string) error {
	log.Printf("Loading %d playlist files into database...", len(m3uFiles))

	for _, filePath := range m3uFiles {
		playlistName := strings.TrimSuffix(filepath.Base(filePath), ".m3u")

		// Parse m3u file
		trackPaths, err := parseM3UFile(filePath)
		if err != nil {
			log.Printf("Error parsing playlist %s: %v", filePath, err)
			continue
		}

		// Create or find playlist
		playlistID, err := createOrFindPlaylist(playlistName, filePath)
		if err != nil {
			log.Printf("Error creating/finding playlist %s: %v", playlistName, err)
			continue
		}

		// Load tracks into playlist
		if err := loadPlaylistTracks(playlistID, trackPaths); err != nil {
			log.Printf("Error loading tracks for playlist %s: %v", playlistName, err)
			continue
		}

		log.Printf("Loaded playlist %s with %d tracks", playlistName, len(trackPaths))
	}

	return nil
}

func writePlaylistToFile(playlistID, filePath string) error {
	// Get playlist tracks in order with metadata for EXTINF lines
	rows, err := db.Query(`
		SELECT s.file_path, s.title, s.artist, COALESCE(s.duration, -1)
		FROM playlist_songs ps
		JOIN songs s ON ps.song_id = s.id
		WHERE ps.playlist_id = $1
		ORDER BY ps.position
	`, playlistID)
	if err != nil {
		return fmt.Errorf("failed to query playlist tracks: %v", err)
	}
	defer rows.Close()

	// Create m3u content
	var content strings.Builder
	content.WriteString("#EXTM3U\n")

	// Get the directory of the m3u file for relative path calculation
	playlistDir := filepath.Dir(filePath)

	for rows.Next() {
		var trackPath, title, artist string
		var duration int
		if err := rows.Scan(&trackPath, &title, &artist, &duration); err != nil {
			log.Printf("Warning: Failed to scan track: %v", err)
			continue
		}
		content.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", duration, artist, title))

		// Convert absolute track path to relative path from the playlist directory
		relativePath, err := filepath.Rel(playlistDir, trackPath)
		if err != nil {
			log.Printf("Warning: Failed to compute relative path for %s: %v, using absolute path", trackPath, err)
			relativePath = trackPath
		}
		content.WriteString(relativePath + "\n")
	}

	// Write to file
	return os.WriteFile(filePath, []byte(content.String()), 0644)
}

// Legacy loadPlaylist function removed - incompatible with database-only approach

func filterTracks(tracks []Track, search string) []Track {
	search = strings.ToLower(search)
	filtered := []Track{}

	for _, track := range tracks {
		if strings.Contains(strings.ToLower(track.Title), search) ||
			strings.Contains(strings.ToLower(track.Artist), search) ||
			strings.Contains(strings.ToLower(track.Album), search) ||
			strings.Contains(strings.ToLower(track.Filename), search) {
			filtered = append(filtered, track)
		}
	}

	return filtered
}

func filterArtists(artists []Artist, search string) []Artist {
	search = strings.ToLower(search)
	filtered := []Artist{}
	for _, artist := range artists {
		if strings.Contains(strings.ToLower(artist.Name), search) {
			filtered = append(filtered, artist)
		}
	}
	return filtered
}

func createPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	var req CreatePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate playlist name
	if req.Name == "" {
		http.Error(w, "Playlist name is required", http.StatusBadRequest)
		return
	}

	// Don't allow the user to use path traversal in the name
	if strings.Contains(req.Name, "..") {
		http.Error(w, "Invalid playlist name", http.StatusBadRequest)
		return
	}

	// Determine the playlist file path based on location
	var playlistPath string
	switch req.Location {
	case "music":
		playlistPath = filepath.Join(musicDir, req.Name+".m3u")
	case "playlists":
		playlistPath = filepath.Join(playlistsDir, req.Name+".m3u")
	case "custom":
		if req.CustomPath == "" {
			http.Error(w, "Custom path is required when location is custom", http.StatusBadRequest)
			return
		}
		if strings.Contains(req.CustomPath, "..") {
			http.Error(w, "Invalid custom path", http.StatusBadRequest)
			return
		}

		// Ensure the custom directory exists
		customDir := filepath.Join(playlistsDir, filepath.Clean(req.CustomPath))
		if err := os.MkdirAll(customDir, 0755); err != nil {
			http.Error(w, "Failed to create custom directory", http.StatusInternalServerError)
			return
		}
		playlistPath = filepath.Join(customDir, req.Name+".m3u")
	default:
		http.Error(w, "Invalid location. Must be 'music', 'playlists', or 'custom'", http.StatusBadRequest)
		return
	}

	// Check if playlist already exists
	if _, err := os.Stat(playlistPath); err == nil {
		http.Error(w, "Playlist already exists", http.StatusConflict)
		return
	}

	// Create empty playlist file with M3U header
	content := "#EXTM3U\n"
	if err := os.WriteFile(playlistPath, []byte(content), 0644); err != nil {
		log.Printf("Error creating playlist file %s: %v", playlistPath, err)
		http.Error(w, "Failed to create playlist file", http.StatusInternalServerError)
		return
	}

	// Add the new playlist to the database
	playlistID := ""
	err := db.QueryRow(`
		INSERT INTO playlists (name, file_path, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`, req.Name, playlistPath, adminUserID).Scan(&playlistID)
	if err != nil {
		log.Printf("Error inserting playlist into database: %v", err)
		http.Error(w, "Failed to create playlist", http.StatusInternalServerError)
		return
	}

	log.Printf("Created playlist %s at %s", req.Name, playlistPath)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Playlist created successfully"))
}

func artistTracksHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	artistName := vars["name"]

	// Query artist tracks from database
	query := `
		SELECT id, title, artist, album, year, file_path, cover_art_id, duration
		FROM songs
		WHERE artist = $1
		ORDER BY album, title
	`

	rows, err := db.Query(query, artistName)
	if err != nil {
		http.Error(w, "Error querying artist tracks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var track Track
		var coverArtID sql.NullString
		var year sql.NullInt32
		var duration sql.NullInt32

		err := rows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
		if err != nil {
			http.Error(w, "Error scanning track", http.StatusInternalServerError)
			return
		}

		if coverArtID.Valid {
			track.CoverArtID = coverArtID.String
		}
		if year.Valid {
			track.Year = int(year.Int32)
		}
		if duration.Valid {
			track.DurationSec = int(duration.Int32)
		}

		tracks = append(tracks, track)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

func albumTracksHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	artistName := vars["artist"]
	albumName := vars["name"]

	// Query album tracks from database
	query := `
		SELECT id, title, artist, album, year, file_path, cover_art_id, duration
		FROM songs
		WHERE artist = $1 AND album = $2
		ORDER BY title
	`

	rows, err := db.Query(query, artistName, albumName)
	if err != nil {
		http.Error(w, "Error querying album tracks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var track Track
		var coverArtID sql.NullString
		var year sql.NullInt32
		var duration sql.NullInt32

		err := rows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
		if err != nil {
			http.Error(w, "Error scanning track", http.StatusInternalServerError)
			return
		}

		if coverArtID.Valid {
			track.CoverArtID = coverArtID.String
		}
		if year.Valid {
			track.Year = int(year.Int32)
		}
		if duration.Valid {
			track.DurationSec = int(duration.Int32)
		}

		tracks = append(tracks, track)
	}

	if len(tracks) == 0 {
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// API Response structures
type APIResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
	Search     string      `json:"search"`
	Total      int         `json:"total"`
}

// Frontend handler
func apiCoverArtHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	coverArtID := vars["id"]

	if coverArtID == "" {
		http.Error(w, "Cover art ID is required", http.StatusBadRequest)
		return
	}

	// Get cover art from database
	content, mimeType, err := getCoverArt(coverArtID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Error retrieving cover art", http.StatusInternalServerError)
		}
		return
	}

	// Set proper headers for caching
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Header().Set("ETag", fmt.Sprintf(`"%s"`, coverArtID))

	// Check if client has cached version
	if match := r.Header.Get("If-None-Match"); match == fmt.Sprintf(`"%s"`, coverArtID) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Write(content)
}

func frontendHandler(w http.ResponseWriter, r *http.Request) {
	// Serve the Vue.js index.html file
	http.ServeFile(w, r, "./web/music/frontend/dist/index.html")
}

// JSON API Handlers

func apiTrackByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trackID := vars["id"]

	var track Track
	var coverArtID *string
	var year *int
	var duration *int

	err := db.QueryRow(`
		SELECT id, title, artist, album, year, file_path, cover_art_id, duration
		FROM songs
		WHERE id = $1
	`, trackID).Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Track not found", http.StatusNotFound)
		} else {
			log.Printf("Error querying track %s: %v", trackID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if year != nil {
		track.Year = *year
	}
	if coverArtID != nil {
		track.CoverArtID = *coverArtID
	}
	if duration != nil {
		track.DurationSec = *duration
		if track.DurationSec > 0 {
			minutes := track.DurationSec / 60
			seconds := track.DurationSec % 60
			track.Duration = fmt.Sprintf("%d:%02d", minutes, seconds)
		}
	}
	track.Filename = filepath.Base(track.FilePath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)
}

func apiTracksHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page == 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	search := r.URL.Query().Get("search")

	// Build SQL query with optional search
	var query string
	var args []interface{}

	if search != "" {
		query = `
			SELECT COUNT(*) FROM songs 
			WHERE title ILIKE $1 OR artist ILIKE $1 OR album ILIKE $1
		`
		args = []interface{}{"%" + search + "%"}
	} else {
		query = "SELECT COUNT(*) FROM songs"
	}

	// Get total count
	var totalCount int
	err := db.QueryRow(query, args...).Scan(&totalCount)
	if err != nil {
		log.Printf("Error counting tracks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	var totalPages int
	if pageSize == 0 {
		// Return all tracks
		pageSize = totalCount
		totalPages = 1
	} else {
		totalPages = (totalCount + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
	}

	// Build tracks query
	offset := (page - 1) * pageSize
	if search != "" {
		query = `
			SELECT id, title, artist, album, year, file_path, cover_art_id, duration
			FROM songs 
			WHERE title ILIKE $1 OR artist ILIKE $1 OR album ILIKE $1
			ORDER BY title
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{"%" + search + "%", pageSize, offset}
	} else {
		query = `
			SELECT id, title, artist, album, year, file_path, cover_art_id, duration
			FROM songs 
			ORDER BY title
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{pageSize, offset}
	}

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying tracks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var track Track
		var coverArtID *string
		var year *int
		var duration *int

		err := rows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
		if err != nil {
			log.Printf("Error scanning track: %v", err)
			continue
		}

		if year != nil {
			track.Year = *year
		}
		if coverArtID != nil {
			track.CoverArtID = *coverArtID
		}
		if duration != nil {
			track.DurationSec = *duration
			if track.DurationSec > 0 {
				minutes := track.DurationSec / 60
				seconds := track.DurationSec % 60
				track.Duration = fmt.Sprintf("%d:%02d", minutes, seconds)
			}
		}

		track.Filename = filepath.Base(track.FilePath)
		tracks = append(tracks, track)
	}

	response := APIResponse{
		Data:       tracks,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Search:     search,
		Total:      totalCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func apiSearchHandler(w http.ResponseWriter, r *http.Request) {
	// This can be the same as apiTracksHandler since tracks endpoint handles search
	apiTracksHandler(w, r)
}

func apiPlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT p.id, p.name, p.description, p.cover_art_id, 
		       COALESCE(track_count.count, 0) as track_count
		FROM playlists p
		LEFT JOIN (
			SELECT playlist_id, COUNT(*) as count
			FROM playlist_songs
			GROUP BY playlist_id
		) track_count ON p.id = track_count.playlist_id
		ORDER BY p.name
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Error querying playlists", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var playlists []Playlist
	for rows.Next() {
		var playlist Playlist
		var description sql.NullString
		var trackCount int
		var coverArtID sql.NullString

		err := rows.Scan(&playlist.ID, &playlist.Name, &description, &coverArtID, &trackCount)
		if err != nil {
			http.Error(w, "Error scanning playlist", http.StatusInternalServerError)
			return
		}

		if description.Valid {
			playlist.Description = description.String
		}
		if coverArtID.Valid {
			playlist.CoverArtID = coverArtID.String
		}

		playlist.TrackCount = trackCount

		playlists = append(playlists, playlist)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlists)
}

func apiCreatePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	var req CreatePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate playlist name
	if req.Name == "" {
		http.Error(w, "Playlist name is required", http.StatusBadRequest)
		return
	}

	// Don't allow the user to use path traversal in the name
	if strings.Contains(req.Name, "..") {
		http.Error(w, "Invalid playlist name", http.StatusBadRequest)
		return
	}

	// Determine the playlist file path based on location
	var playlistPath string
	switch req.Location {
	case "music":
		playlistPath = filepath.Join(musicDir, req.Name+".m3u")
	case "playlists":
		playlistPath = filepath.Join(playlistsDir, req.Name+".m3u")
	case "custom":
		if req.CustomPath == "" {
			http.Error(w, "Custom path is required when location is custom", http.StatusBadRequest)
			return
		}

		// Don't allow path traversal
		if strings.Contains(req.CustomPath, "..") {
			http.Error(w, "Invalid custom path", http.StatusBadRequest)
			return
		}

		// Ensure the custom directory exists
		customDir := filepath.Join(playlistsDir, filepath.Clean(req.CustomPath))
		if err := os.MkdirAll(customDir, 0755); err != nil {
			http.Error(w, "Failed to create custom directory", http.StatusInternalServerError)
			return
		}
		playlistPath = filepath.Join(customDir, req.Name+".m3u")
	default:
		http.Error(w, "Invalid location. Must be 'music', 'playlists', or 'custom'", http.StatusBadRequest)
		return
	}

	// Check if playlist already exists
	if _, err := os.Stat(playlistPath); err == nil {
		http.Error(w, "Playlist already exists", http.StatusConflict)
		return
	}

	// Create empty playlist file with M3U header
	content := "#EXTM3U\n"
	if err := os.WriteFile(playlistPath, []byte(content), 0644); err != nil {
		log.Printf("Error creating playlist file %s: %v", playlistPath, err)
		http.Error(w, "Failed to create playlist file", http.StatusInternalServerError)
		return
	}

	// Add the new playlist to the database
	playlistID := ""
	err := db.QueryRow(`
		INSERT INTO playlists (name, file_path, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`, req.Name, playlistPath, adminUserID).Scan(&playlistID)
	if err != nil {
		log.Printf("Error inserting playlist into database: %v", err)
		http.Error(w, "Failed to create playlist", http.StatusInternalServerError)
		return
	}

	log.Printf("Created playlist %s at %s", req.Name, playlistPath)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Playlist created successfully"})
}

// New ID-based playlist handlers
func apiPlaylistByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	// Get playlist from database
	var playlist Playlist
	var description sql.NullString
	var filePath sql.NullString
	var coverArtID sql.NullString

	err := db.QueryRow("SELECT id, name, description, file_path, cover_art_id FROM playlists WHERE id = $1", playlistID).Scan(&playlist.ID, &playlist.Name, &description, &filePath, &coverArtID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error querying playlist", http.StatusInternalServerError)
		}
		return
	}

	if description.Valid {
		playlist.Description = description.String
	}
	if filePath.Valid {
		playlist.Path = filePath.String
	}
	if coverArtID.Valid {
		playlist.CoverArtID = coverArtID.String
	}

	// Count total tracks
	var totalTracks int

	err = db.QueryRow("SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1", playlist.ID).Scan(&totalTracks)
	if err != nil {
		http.Error(w, "Error counting playlist tracks", http.StatusInternalServerError)
		return
	}

	// var tracks []Track
	// for rows.Next() {
	// 	var track Track
	// 	var coverArtID *string
	// 	var year *int
	// 	var duration *int

	// 	err := rows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
	// 	if err != nil {
	// 		log.Printf("Error scanning track: %v", err)
	// 		continue
	// 	}

	// 	if year != nil {
	// 		track.Year = *year
	// 	}
	// 	if coverArtID != nil {
	// 		track.CoverArtID = *coverArtID
	// 	}
	// 	if duration != nil {
	// 		track.DurationSec = *duration
	// 		if track.DurationSec > 0 {
	// 			minutes := track.DurationSec / 60
	// 			seconds := track.DurationSec % 60
	// 			track.Duration = fmt.Sprintf("%d:%02d", minutes, seconds)
	// 		}
	// 	}

	// 	track.Filename = filepath.Base(track.FilePath)
	// 	tracks = append(tracks, track)
	// }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlist)
}

func apiPlaylistTracksByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	// Parse query parameters
	searchQuery := r.URL.Query().Get("search")
	page := 1
	pageSize := 50

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}

	// Verify playlist exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM playlists WHERE id = $1)", playlistID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}

	// Count total tracks
	var totalTracks int
	var countQuery string
	var countArgs []interface{}

	if searchQuery != "" {
		countQuery = `
			SELECT COUNT(*)
			FROM playlist_songs ps
			JOIN songs s ON ps.song_id = s.id
			WHERE ps.playlist_id = $1 AND (s.title ILIKE $2 OR s.artist ILIKE $2 OR s.album ILIKE $2)
		`
		countArgs = []interface{}{playlistID, "%" + searchQuery + "%"}
	} else {
		countQuery = "SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1"
		countArgs = []interface{}{playlistID}
	}

	err = db.QueryRow(countQuery, countArgs...).Scan(&totalTracks)
	if err != nil {
		http.Error(w, "Error counting playlist tracks", http.StatusInternalServerError)
		return
	}

	totalPages := (totalTracks + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize

	// Get playlist tracks with pagination
	var query string
	var args []interface{}

	if searchQuery != "" {
		query = `
			SELECT s.id, s.title, s.artist, s.album, s.year, s.file_path, s.cover_art_id, s.duration
			FROM playlist_songs ps
			JOIN songs s ON ps.song_id = s.id
			WHERE ps.playlist_id = $1 AND (s.title ILIKE $2 OR s.artist ILIKE $2 OR s.album ILIKE $2)
			ORDER BY ps.position
			LIMIT $3 OFFSET $4
		`
		args = []interface{}{playlistID, "%" + searchQuery + "%", pageSize, offset}
	} else {
		query = `
			SELECT s.id, s.title, s.artist, s.album, s.year, s.file_path, s.cover_art_id, s.duration
			FROM playlist_songs ps
			JOIN songs s ON ps.song_id = s.id
			WHERE ps.playlist_id = $1
			ORDER BY ps.position
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{playlistID, pageSize, offset}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Error querying playlist tracks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var track Track
		var coverArtID *string
		var year *int
		var duration *int

		err := rows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
		if err != nil {
			log.Printf("Error scanning track: %v", err)
			continue
		}

		if year != nil {
			track.Year = *year
		}
		if coverArtID != nil {
			track.CoverArtID = *coverArtID
		}
		if duration != nil {
			track.DurationSec = *duration
			if track.DurationSec > 0 {
				minutes := track.DurationSec / 60
				seconds := track.DurationSec % 60
				track.Duration = fmt.Sprintf("%d:%02d", minutes, seconds)
			}
		}

		track.Filename = filepath.Base(track.FilePath)
		tracks = append(tracks, track)
	}

	response := map[string]interface{}{
		"data":       tracks,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
		"search":     searchQuery,
		"total":      totalTracks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func apiAddToPlaylistByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]
	trackID := vars["trackId"]

	log.Printf("Adding track %s to playlist %s", trackID, playlistID)

	// Verify playlist exists and get file path
	var playlistName string
	var filePath sql.NullString
	err := db.QueryRow("SELECT name, file_path FROM playlists WHERE id = $1", playlistID).Scan(&playlistName, &filePath)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error querying playlist", http.StatusInternalServerError)
		}
		return
	}

	// Verify track exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM songs WHERE id = $1)", trackID).Scan(&exists)
	if err != nil {
		http.Error(w, "Error querying track", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	// Check if track is already in playlist
	var alreadyExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2)", playlistID, trackID).Scan(&alreadyExists)
	if err != nil {
		http.Error(w, "Error checking playlist", http.StatusInternalServerError)
		return
	}

	if alreadyExists {
		http.Error(w, "Track already in playlist", http.StatusConflict)
		return
	}

	// Get the next position for this playlist
	var nextPosition int
	err = db.QueryRow("SELECT COALESCE(MAX(position), 0) + 1 FROM playlist_songs WHERE playlist_id = $1", playlistID).Scan(&nextPosition)
	if err != nil {
		http.Error(w, "Error getting next position", http.StatusInternalServerError)
		return
	}

	// Add track to playlist in database
	_, err = db.Exec("INSERT INTO playlist_songs (playlist_id, song_id, position) VALUES ($1, $2, $3)", playlistID, trackID, nextPosition)
	if err != nil {
		http.Error(w, "Error adding track to playlist", http.StatusInternalServerError)
		return
	}

	// Update m3u file if file path exists
	if filePath.Valid && filePath.String != "" {
		if err := writePlaylistToFile(playlistID, filePath.String); err != nil {
			log.Printf("Warning: Failed to update playlist file %s: %v", filePath.String, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Track added to playlist"})
}

func apiRemoveFromPlaylistByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]
	trackID := vars["trackId"]

	log.Printf("Removing track %s from playlist %s", trackID, playlistID)

	// Verify playlist exists and get file path
	var playlistName string
	var filePath sql.NullString
	err := db.QueryRow("SELECT name, file_path FROM playlists WHERE id = $1", playlistID).Scan(&playlistName, &filePath)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error querying playlist", http.StatusInternalServerError)
		}
		return
	}

	// Remove track from playlist
	result, err := db.Exec("DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2", playlistID, trackID)
	if err != nil {
		http.Error(w, "Error removing track from playlist", http.StatusInternalServerError)
		return
	}

	// Check if anything was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error checking deletion result", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Track not found in playlist", http.StatusNotFound)
		return
	}

	// Update m3u file if file path exists
	if filePath.Valid && filePath.String != "" {
		if err := writePlaylistToFile(playlistID, filePath.String); err != nil {
			log.Printf("Warning: Failed to update playlist file %s: %v", filePath.String, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Track removed from playlist"})
}

func apiDeletePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	log.Printf("Deleting playlist %s", playlistID)

	// Get playlist info (name and file path) before deleting
	var playlistName string
	var filePath sql.NullString
	err := db.QueryRow("SELECT name, file_path FROM playlists WHERE id = $1", playlistID).Scan(&playlistName, &filePath)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Playlist not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error querying playlist", http.StatusInternalServerError)
		}
		return
	}

	// Remove all tracks from the playlist
	_, err = db.Exec("DELETE FROM playlist_songs WHERE playlist_id = $1", playlistID)
	if err != nil {
		http.Error(w, "Error removing playlist tracks", http.StatusInternalServerError)
		return
	}

	// Remove the playlist itself
	_, err = db.Exec("DELETE FROM playlists WHERE id = $1", playlistID)
	if err != nil {
		http.Error(w, "Error deleting playlist", http.StatusInternalServerError)
		return
	}

	// Delete the m3u file from disk
	if filePath.Valid && filePath.String != "" {
		if err := os.Remove(filePath.String); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to delete playlist file %s: %v", filePath.String, err)
		}
	}

	log.Printf("Deleted playlist %s (%s)", playlistName, playlistID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Playlist deleted"})
}

func apiScanStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	scanning := atomic.LoadInt32(&isScanning) == 1
	json.NewEncoder(w).Encode(map[string]bool{"scanning": scanning})
}

func apiRescanHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to set isScanning flag; if already scanning, return 409 Conflict
	if !atomic.CompareAndSwapInt32(&isScanning, 0, 1) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"message": "Scan already in progress"})
		return
	}

	go func() {
		defer atomic.StoreInt32(&isScanning, 0)
		log.Printf("Starting library rescan...")
		if err := scanMusicLibrary(); err != nil {
			log.Printf("Error during rescan: %v", err)
		} else {
			log.Printf("Library rescanned successfully")
		}
	}()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Rescan started"})
}

func apiArtistsHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page == 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	search := r.URL.Query().Get("search")

	// Build SQL query for total count
	var countQuery string
	var args []interface{}
	if search != "" {
		countQuery = "SELECT COUNT(*) FROM artists WHERE name ILIKE $1"
		args = []interface{}{"%" + search + "%"}
	} else {
		countQuery = "SELECT COUNT(*) FROM artists"
	}

	// Get total count
	var totalCount int
	err := db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		http.Error(w, "Error counting artists", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	var totalPages int
	if pageSize == 0 {
		totalPages = 1
		pageSize = totalCount // Return all artists
	} else {
		totalPages = (totalCount + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
	}

	// Build main query with pagination
	var query string
	offset := (page - 1) * pageSize
	if search != "" {
		query = `
			SELECT a.id, a.name, a.cover_art_id,
			       COALESCE(album_count.count, 0) as album_count,
			       COALESCE(track_count.count, 0) as track_count
			FROM artists a
			LEFT JOIN (
				SELECT artist_id, COUNT(*) as count
				FROM albums
				GROUP BY artist_id
			) album_count ON a.id = album_count.artist_id
			LEFT JOIN (
				SELECT artist, COUNT(*) as count
				FROM songs
				GROUP BY artist
			) track_count ON a.name = track_count.artist
			WHERE a.name ILIKE $1
			ORDER BY a.name
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{"%" + search + "%", pageSize, offset}
	} else {
		query = `
			SELECT a.id, a.name, a.cover_art_id,
			       COALESCE(album_count.count, 0) as album_count,
			       COALESCE(track_count.count, 0) as track_count
			FROM artists a
			LEFT JOIN (
				SELECT artist_id, COUNT(*) as count
				FROM albums
				GROUP BY artist_id
			) album_count ON a.id = album_count.artist_id
			LEFT JOIN (
				SELECT artist, COUNT(*) as count
				FROM songs
				GROUP BY artist
			) track_count ON a.name = track_count.artist
			ORDER BY a.name
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{pageSize, offset}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Error querying artists", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pageArtists []Artist
	for rows.Next() {
		var artist Artist
		var coverArtID sql.NullString
		var albumCount, trackCount int

		err := rows.Scan(&artist.ID, &artist.Name, &coverArtID, &albumCount, &trackCount)
		if err != nil {
			http.Error(w, "Error scanning artist", http.StatusInternalServerError)
			return
		}

		if coverArtID.Valid {
			artist.CoverArtID = coverArtID.String
		}

		// Get albums for this artist
		albumQuery := `
			SELECT id, title, year, cover_art_id
			FROM albums
			WHERE artist_id = $1
			ORDER BY year, title
		`
		albumRows, err := db.Query(albumQuery, artist.ID)
		if err != nil {
			http.Error(w, "Error querying albums", http.StatusInternalServerError)
			return
		}

		for albumRows.Next() {
			var album Album
			var albumCoverArtID sql.NullString
			var year sql.NullInt32

			err := albumRows.Scan(&album.ID, &album.Name, &year, &albumCoverArtID)
			if err != nil {
				albumRows.Close()
				http.Error(w, "Error scanning album", http.StatusInternalServerError)
				return
			}

			if albumCoverArtID.Valid {
				album.CoverArtID = albumCoverArtID.String
			}
			if year.Valid {
				album.Year = int(year.Int32)
			}
			album.Artist = artist.Name

			// Get track count for album
			var albumTrackCount int
			err = db.QueryRow("SELECT COUNT(*) FROM songs WHERE album = $1 AND artist = $2", album.Name, artist.Name).Scan(&albumTrackCount)
			if err != nil {
				albumRows.Close()
				http.Error(w, "Error counting album tracks", http.StatusInternalServerError)
				return
			}

			// Create tracks array with correct length but don't populate (for performance)
			album.Tracks = make([]Track, albumTrackCount)

			artist.Albums = append(artist.Albums, &album)
		}
		albumRows.Close()

		// Create tracks array with correct length but don't populate (for performance)
		artist.Tracks = make([]Track, trackCount)

		pageArtists = append(pageArtists, artist)
	}

	response := APIResponse{
		Data:       pageArtists,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Search:     search,
		Total:      totalCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ID-based handlers for safer URL handling
func apiArtistByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	artistID := vars["id"]

	// Query artist directly from database
	var artist Artist
	var coverArtID *string

	err := db.QueryRow(`
		SELECT id, name, cover_art_id
		FROM artists
		WHERE id = $1
	`, artistID).Scan(&artist.ID, &artist.Name, &coverArtID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Artist not found", http.StatusNotFound)
		} else {
			log.Printf("Error querying artist: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if coverArtID != nil {
		artist.CoverArtID = *coverArtID
	}

	// Get albums for this artist from database
	albumRows, err := db.Query(`
		SELECT id, title, year, cover_art_id
		FROM albums
		WHERE artist_id = $1
		ORDER BY title
	`, artistID)

	if err != nil {
		log.Printf("Error querying artist albums: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer albumRows.Close()

	var albums []*Album
	for albumRows.Next() {
		var album Album
		var year *int
		var coverArtID *string

		err := albumRows.Scan(&album.ID, &album.Name, &year, &coverArtID)
		if err != nil {
			log.Printf("Error scanning album: %v", err)
			continue
		}

		if year != nil {
			album.Year = *year
		}
		if coverArtID != nil {
			album.CoverArtID = *coverArtID
		}

		album.Artist = artist.Name
		albums = append(albums, &album)
	}

	// Get tracks for this artist from database and populate each album's tracks
	trackRows, err := db.Query(`
		SELECT id, title, artist, album, year, file_path, cover_art_id, duration
		FROM songs
		WHERE artist = $1
		ORDER BY album, title
	`, artist.Name)

	if err != nil {
		log.Printf("Error querying artist tracks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer trackRows.Close()

	var tracks []Track
	albumTracksMap := make(map[string][]Track) // Map album name to tracks

	for trackRows.Next() {
		var track Track
		var coverArtID *string
		var year *int
		var duration *int

		err := trackRows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
		if err != nil {
			log.Printf("Error scanning track: %v", err)
			continue
		}

		if year != nil {
			track.Year = *year
		}
		if coverArtID != nil {
			track.CoverArtID = *coverArtID
		}
		if duration != nil {
			track.DurationSec = *duration
			if track.DurationSec > 0 {
				minutes := track.DurationSec / 60
				seconds := track.DurationSec % 60
				track.Duration = fmt.Sprintf("%d:%02d", minutes, seconds)
			}
		}

		track.Filename = filepath.Base(track.FilePath)
		tracks = append(tracks, track)

		// Add track to the album's track list
		albumTracksMap[track.Album] = append(albumTracksMap[track.Album], track)
	}

	// Populate tracks for each album
	for _, album := range albums {
		if albumTracks, exists := albumTracksMap[album.Name]; exists {
			album.Tracks = albumTracks
		}
	}

	artist.Albums = albums
	artist.Tracks = tracks

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page == 0 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize == 0 {
		pageSize = 25
	}

	searchQuery := r.URL.Query().Get("q")

	tracks = artist.Tracks
	if searchQuery != "" {
		var filteredTracks []Track
		searchLower := strings.ToLower(searchQuery)
		for _, track := range tracks {
			if strings.Contains(strings.ToLower(track.Title), searchLower) ||
				strings.Contains(strings.ToLower(track.Album), searchLower) {
				filteredTracks = append(filteredTracks, track)
			}
		}
		tracks = filteredTracks
	}

	totalPages := (len(tracks) + pageSize - 1) / pageSize
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(tracks) {
		endIdx = len(tracks)
	}

	var pagedTracks []Track
	if startIdx < len(tracks) {
		pagedTracks = tracks[startIdx:endIdx]
	}

	response := map[string]interface{}{
		"artist":     artist,
		"data":       pagedTracks,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
		"search":     searchQuery,
		"total":      len(tracks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func apiArtistTracksByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	artistID := vars["id"]

	// Parse query parameters
	searchQuery := r.URL.Query().Get("search")
	page := 1
	pageSize := 50

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}

	// First verify artist exists and get name
	var artistName string
	err := db.QueryRow("SELECT name FROM artists WHERE id = $1", artistID).Scan(&artistName)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Artist not found", http.StatusNotFound)
		} else {
			log.Printf("Error querying artist: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Count total tracks
	var totalTracks int
	var countQuery string
	var countArgs []interface{}

	if searchQuery != "" {
		countQuery = `
			SELECT COUNT(*)
			FROM songs
			WHERE artist = $1 AND (title ILIKE $2 OR album ILIKE $2)
		`
		countArgs = []interface{}{artistName, "%" + searchQuery + "%"}
	} else {
		countQuery = `
			SELECT COUNT(*)
			FROM songs
			WHERE artist = $1
		`
		countArgs = []interface{}{artistName}
	}

	err = db.QueryRow(countQuery, countArgs...).Scan(&totalTracks)
	if err != nil {
		log.Printf("Error counting tracks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	totalPages := (totalTracks + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize

	// Get tracks for this artist with pagination
	var query string
	var args []interface{}

	if searchQuery != "" {
		query = `
			SELECT id, title, artist, album, year, file_path, cover_art_id, duration
			FROM songs
			WHERE artist = $1 AND (title ILIKE $2 OR album ILIKE $2)
			ORDER BY album, title
			LIMIT $3 OFFSET $4
		`
		args = []interface{}{artistName, "%" + searchQuery + "%", pageSize, offset}
	} else {
		query = `
			SELECT id, title, artist, album, year, file_path, cover_art_id, duration
			FROM songs
			WHERE artist = $1
			ORDER BY album, title
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{artistName, pageSize, offset}
	}

	trackRows, err := db.Query(query, args...)

	if err != nil {
		log.Printf("Error querying artist tracks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer trackRows.Close()

	var tracks []Track
	for trackRows.Next() {
		var track Track
		var coverArtID *string
		var year *int
		var duration *int

		err := trackRows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
		if err != nil {
			log.Printf("Error scanning track: %v", err)
			continue
		}

		if year != nil {
			track.Year = *year
		}
		if coverArtID != nil {
			track.CoverArtID = *coverArtID
		}
		if duration != nil {
			track.DurationSec = *duration
			if track.DurationSec > 0 {
				minutes := track.DurationSec / 60
				seconds := track.DurationSec % 60
				track.Duration = fmt.Sprintf("%d:%02d", minutes, seconds)
			}
		}

		track.Filename = filepath.Base(track.FilePath)
		tracks = append(tracks, track)
	}

	response := map[string]interface{}{
		"data":       tracks,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
		"search":     searchQuery,
		"total":      totalTracks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func apiAlbumByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	albumID := vars["id"]

	log.Printf("DEBUG: apiAlbumByIDHandler called with albumID: %s", albumID)

	// Query album directly from database
	var album Album
	var artistName string
	var year *int
	var coverArtID *string

	err := db.QueryRow(`
		SELECT a.id, a.title, a.year, a.cover_art_id, ar.name as artist_name
		FROM albums a
		JOIN artists ar ON a.artist_id = ar.id
		WHERE a.id = $1
	`, albumID).Scan(&album.ID, &album.Name, &year, &coverArtID, &artistName)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("DEBUG: Album not found with ID: %s", albumID)
			http.Error(w, "Album not found", http.StatusNotFound)
		} else {
			log.Printf("Error querying album with ID %s: %v", albumID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("DEBUG: Found album: %s by %s", album.Name, artistName)

	if year != nil {
		album.Year = *year
	}
	if coverArtID != nil {
		album.CoverArtID = *coverArtID
	}
	album.Artist = artistName

	// Get tracks for this album from database
	trackRows, err := db.Query(`
		SELECT id, title, artist, album, year, file_path, cover_art_id, duration
		FROM songs 
		WHERE artist = $1 AND album = $2
		ORDER BY title
	`, artistName, album.Name)

	if err != nil {
		log.Printf("Error querying album tracks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer trackRows.Close()

	var tracks []Track
	for trackRows.Next() {
		var track Track
		var coverArtID *string
		var year *int
		var duration *int

		err := trackRows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
		if err != nil {
			log.Printf("Error scanning track: %v", err)
			continue
		}

		if year != nil {
			track.Year = *year
		}
		if coverArtID != nil {
			track.CoverArtID = *coverArtID
		}
		if duration != nil {
			track.DurationSec = *duration
			if track.DurationSec > 0 {
				minutes := track.DurationSec / 60
				seconds := track.DurationSec % 60
				track.Duration = fmt.Sprintf("%d:%02d", minutes, seconds)
			}
		}

		tracks = append(tracks, track)
	}

	album.Tracks = tracks

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page == 0 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize == 0 {
		pageSize = 25
	}

	searchQuery := r.URL.Query().Get("q")

	tracks = album.Tracks
	if searchQuery != "" {
		var filteredTracks []Track
		searchLower := strings.ToLower(searchQuery)
		for _, track := range tracks {
			if strings.Contains(strings.ToLower(track.Title), searchLower) ||
				strings.Contains(strings.ToLower(track.Artist), searchLower) {
				filteredTracks = append(filteredTracks, track)
			}
		}
		tracks = filteredTracks
	}

	totalPages := (len(tracks) + pageSize - 1) / pageSize
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(tracks) {
		endIdx = len(tracks)
	}

	var pagedTracks []Track
	if startIdx < len(tracks) {
		pagedTracks = tracks[startIdx:endIdx]
	}

	response := map[string]interface{}{
		"album":      album,
		"artist":     album.Artist,
		"data":       pagedTracks,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
		"search":     searchQuery,
		"total":      len(tracks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func apiAlbumTracksByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	albumID := vars["id"]

	// Parse query parameters
	searchQuery := r.URL.Query().Get("search")
	page := 1
	pageSize := 50

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}

	// First verify album exists and get its details
	var albumName string
	var artistName string
	err := db.QueryRow(`
		SELECT a.title, ar.name
		FROM albums a
		JOIN artists ar ON a.artist_id = ar.id
		WHERE a.id = $1
	`, albumID).Scan(&albumName, &artistName)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Album not found", http.StatusNotFound)
		} else {
			log.Printf("Error querying album: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Count total tracks
	var totalTracks int
	var countQuery string
	var countArgs []interface{}

	if searchQuery != "" {
		countQuery = `
			SELECT COUNT(*)
			FROM songs
			WHERE artist = $1 AND album = $2 AND title ILIKE $3
		`
		countArgs = []interface{}{artistName, albumName, "%" + searchQuery + "%"}
	} else {
		countQuery = `
			SELECT COUNT(*)
			FROM songs
			WHERE artist = $1 AND album = $2
		`
		countArgs = []interface{}{artistName, albumName}
	}

	err = db.QueryRow(countQuery, countArgs...).Scan(&totalTracks)
	if err != nil {
		log.Printf("Error counting tracks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	totalPages := (totalTracks + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize

	// Get tracks for this album with pagination
	var query string
	var args []interface{}

	if searchQuery != "" {
		query = `
			SELECT id, title, artist, album, year, file_path, cover_art_id, duration
			FROM songs
			WHERE artist = $1 AND album = $2 AND title ILIKE $3
			ORDER BY title
			LIMIT $4 OFFSET $5
		`
		args = []interface{}{artistName, albumName, "%" + searchQuery + "%", pageSize, offset}
	} else {
		query = `
			SELECT id, title, artist, album, year, file_path, cover_art_id, duration
			FROM songs
			WHERE artist = $1 AND album = $2
			ORDER BY title
			LIMIT $3 OFFSET $4
		`
		args = []interface{}{artistName, albumName, pageSize, offset}
	}

	trackRows, err := db.Query(query, args...)

	if err != nil {
		log.Printf("Error querying album tracks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer trackRows.Close()

	var tracks []Track
	for trackRows.Next() {
		var track Track
		var coverArtID *string
		var year *int
		var duration *int

		err := trackRows.Scan(&track.ID, &track.Title, &track.Artist, &track.Album, &year, &track.FilePath, &coverArtID, &duration)
		if err != nil {
			log.Printf("Error scanning track: %v", err)
			continue
		}

		if year != nil {
			track.Year = *year
		}
		if coverArtID != nil {
			track.CoverArtID = *coverArtID
		}
		if duration != nil {
			track.DurationSec = *duration
			if track.DurationSec > 0 {
				minutes := track.DurationSec / 60
				seconds := track.DurationSec % 60
				track.Duration = fmt.Sprintf("%d:%02d", minutes, seconds)
			}
		}

		track.Filename = filepath.Base(track.FilePath)
		tracks = append(tracks, track)
	}

	response := map[string]interface{}{
		"data":       tracks,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": totalPages,
		"search":     searchQuery,
		"total":      totalTracks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func apiLyricsHandler(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")
	title := r.URL.Query().Get("title")

	if artist == "" || title == "" {
		http.Error(w, "Artist and title parameters are required", http.StatusBadRequest)
		return
	}

	lyrics, err := fetchLyrics(artist, title)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Lyrics not found",
			"message": err.Error(),
		})
		return
	}

	response := map[string]string{
		"lyrics": lyrics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func fetchLyrics(artist, title string) (string, error) {
	// For now, return a simple implementation that returns placeholder lyrics
	// In a real application, you would integrate with a lyrics API service like:
	// - Genius API
	// - MusixMatch API
	// - LyricsFind API

	// Simple placeholder implementation
	return fmt.Sprintf(`No lyrics found for "%s" by %s.

This is a placeholder lyrics service. To implement actual lyrics search, you would need to:

1. Sign up for a lyrics API service (Genius, MusixMatch, etc.)
2. Get an API key
3. Implement the HTTP requests to search for lyrics
4. Parse and return the lyrics text

Example lyrics services:
• Genius API: https://docs.genius.com/
• MusixMatch API: https://developer.musixmatch.com/
• LyricFind API: https://www.lyricfind.com/

For now, this returns placeholder text to demonstrate the UI functionality.`, title, artist), nil
}

// Remote control session handlers

type SessionInfo struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	HasPlayer       bool      `json:"has_player"`
	ControllerCount int       `json:"controller_count"`
	LastSeen        time.Time `json:"last_seen"`
}

func apiSessionsHandler(w http.ResponseWriter, r *http.Request) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	sessions := make([]SessionInfo, 0)
	for _, sess := range hub.sessions {
		sess.mu.Lock()
		hasPlayer := sess.PlayerConn != nil
		controllerCount := len(sess.Controllers)
		sessionName := sess.Name
		sessionID := sess.ID
		sess.mu.Unlock()

		sessions = append(sessions, SessionInfo{
			ID:              sessionID,
			Name:            sessionName,
			HasPlayer:       hasPlayer,
			ControllerCount: controllerCount,
			LastSeen:        time.Now(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func apiPlayerWSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// First message should be register with session_id
	var registerMsg map[string]interface{}
	err = conn.ReadJSON(&registerMsg)
	if err != nil {
		log.Printf("Error reading register message: %v", err)
		return
	}

	sessionID := ""
	if val, ok := registerMsg["session_id"]; ok {
		sessionID, _ = val.(string)
	}
	clientHostname := ""
	if val, ok := registerMsg["client_hostname"]; ok {
		clientHostname, _ = val.(string)
	}
	customSessionName := ""
	if val, ok := registerMsg["session_name"]; ok {
		customSessionName, _ = val.(string)
	}

	hub.mu.Lock()
	var session *PlayerSession
	if sessionID != "" {
		// Try to resume existing session
		if existing, found := hub.sessions[sessionID]; found {
			session = existing
			session.mu.Lock()
			session.PlayerConn = conn
			if session.controllerMus == nil {
				session.controllerMus = make(map[*websocket.Conn]*sync.Mutex)
			}
			session.DisconnectedAt = nil
			// Update name if client provided a new custom name
			if customSessionName != "" {
				session.Name = customSessionName
			}
			session.mu.Unlock()
		}
	}

	if session == nil {
		// Create new session
		sessionID = uuid.New().String()
		// Use custom name, or client hostname + short ID, or server hostname + short ID
		var sessionName string
		if customSessionName != "" {
			sessionName = customSessionName
		} else if clientHostname != "" {
			sessionName = fmt.Sprintf("%s-%s", clientHostname, sessionID[:8])
		} else {
			sessionName = fmt.Sprintf("%s-%s", serverHostname, sessionID[:8])
		}
		session = &PlayerSession{
			ID:            sessionID,
			Name:          sessionName,
			PlayerConn:    conn,
			Controllers:   make([]*websocket.Conn, 0),
			controllerMus: make(map[*websocket.Conn]*sync.Mutex),
		}
		hub.sessions[sessionID] = session
	}
	hub.mu.Unlock()

	// Send session ack
	session.writeToPlayer(map[string]string{
		"type":         "session_ack",
		"session_id":   session.ID,
		"session_name": session.Name,
	})

	// Listen for state updates and broadcast to controllers
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Player WebSocket error: %v", err)
			session.mu.Lock()
			session.PlayerConn = nil
			disconnectTime := time.Now()
			session.DisconnectedAt = &disconnectTime
			session.mu.Unlock()

			// Clean up session after 60s grace period
			go func() {
				time.Sleep(60 * time.Second)
				hub.mu.Lock()
				if existing, found := hub.sessions[sessionID]; found {
					existing.mu.Lock()
					if existing.PlayerConn == nil && existing.DisconnectedAt != nil {
						hub.mu.Unlock()
						existing.mu.Unlock()
						hub.mu.Lock()
						delete(hub.sessions, sessionID)
						hub.mu.Unlock()
						return
					}
					existing.mu.Unlock()
				}
				hub.mu.Unlock()
			}()
			return
		}

		msgType, _ := msg["type"].(string)
		if msgType == "rename" {
			if newName, ok := msg["session_name"].(string); ok && newName != "" {
				session.mu.Lock()
				session.Name = newName
				session.mu.Unlock()
			}
			continue
		}
		if msgType == "state" {
			// Cache state and broadcast to controllers
			stateJSON, _ := json.Marshal(msg)
			session.mu.Lock()
			session.LastState = stateJSON
			session.mu.Unlock()
			session.broadcastToControllers(msg)
		}
	}
}

func apiControlWSHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	hub.mu.RLock()
	session, found := hub.sessions[sessionID]
	hub.mu.RUnlock()

	if !found {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Add controller to session with its own write mutex
	connMu := &sync.Mutex{}
	session.mu.Lock()
	session.Controllers = append(session.Controllers, conn)
	session.controllerMus[conn] = connMu

	// Send cached state immediately (safe: we're the only writer so far)
	if len(session.LastState) > 0 {
		connMu.Lock()
		conn.WriteMessage(websocket.TextMessage, session.LastState)
		connMu.Unlock()
	}

	controllerCount := len(session.Controllers)
	session.mu.Unlock()

	// Notify player of new controller
	controllersUpdate := map[string]interface{}{
		"type":  "controllers_update",
		"count": controllerCount,
	}
	session.writeToPlayer(controllersUpdate)

	// Listen for commands and forward to player
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Controller WebSocket error: %v", err)

			// Remove controller from session (search by pointer, not index)
			session.mu.Lock()
			for i, cc := range session.Controllers {
				if cc == conn {
					session.Controllers = append(session.Controllers[:i], session.Controllers[i+1:]...)
					break
				}
			}
			delete(session.controllerMus, conn)
			newCount := len(session.Controllers)
			session.mu.Unlock()

			// Notify player of controller count change
			session.writeToPlayer(map[string]interface{}{
				"type":  "controllers_update",
				"count": newCount,
			})
			return
		}

		msgType, _ := msg["type"].(string)
		if msgType == "command" {
			// Forward command to player
			session.writeToPlayer(msg)
		}
	}
}
