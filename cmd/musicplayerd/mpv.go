package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// MpvClient manages an mpv subprocess and communicates via JSON IPC over a Unix socket.
type MpvClient struct {
	socketPath  string
	audioDevice string
	volume      float64

	cmd   *exec.Cmd
	conn  net.Conn
	mu    sync.Mutex // serialises writes to the socket
	reqID atomic.Int64

	// Pending request-response correlation.
	pending   map[int64]chan json.RawMessage
	pendingMu sync.Mutex

	// Event callbacks (set by the Daemon before Start).
	OnTimePos  func(float64)
	OnDuration func(float64)
	OnTrackEnd func(reason, fileError string) // reason: "eof", "error", "stop"; fileError: mpv error string
}

// NewMpvClient creates a new mpv IPC client. Call Start() to launch the process.
func NewMpvClient(socketPath, audioDevice string, volume float64) *MpvClient {
	return &MpvClient{
		socketPath:  socketPath,
		audioDevice: audioDevice,
		volume:      volume,
		pending:     make(map[int64]chan json.RawMessage),
	}
}

// Start launches the mpv process in idle mode and connects the IPC socket.
func (m *MpvClient) Start() error {
	// Remove stale socket file.
	os.Remove(m.socketPath)

	args := []string{
		"--idle",
		"--no-video",
		"--no-terminal",
		"--input-ipc-server=" + m.socketPath,
		fmt.Sprintf("--volume=%d", int(m.volume)),
	}

	// Audio output: default to pulse (works with PipeWire too).
	if m.audioDevice != "" && m.audioDevice != "auto" {
		args = append(args, "--ao=pulse", "--audio-device=pulse/"+m.audioDevice)
	} else {
		args = append(args, "--ao=pulse")
	}

	m.cmd = exec.Command("mpv", args...)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("start mpv: %w", err)
	}

	// Wait for the IPC socket to appear.
	if err := m.waitForSocket(5 * time.Second); err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("mpv socket not ready: %w", err)
	}

	conn, err := net.Dial("unix", m.socketPath)
	if err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("connect to mpv socket: %w", err)
	}
	m.conn = conn

	go m.readLoop()

	// Observe properties for position and duration updates.
	m.observeProperty(1, "time-pos")
	m.observeProperty(2, "duration")

	log.Printf("mpv started (pid %d, socket %s)", m.cmd.Process.Pid, m.socketPath)
	return nil
}

// StopPlayback stops the currently playing file without killing mpv.
func (m *MpvClient) Stop() {
	m.sendCommand("stop")
}

// Shutdown kills the mpv process and cleans up.
func (m *MpvClient) Shutdown() {
	if m.conn != nil {
		m.sendCommand("quit")
		m.conn.Close()
	}
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
	}
	os.Remove(m.socketPath)
}

// LoadFile tells mpv to play a file/URL.
func (m *MpvClient) LoadFile(url string) error {
	return m.sendCommand("loadfile", url)
}

// SetPause sets the pause state.
func (m *MpvClient) SetPause(paused bool) error {
	return m.setProperty("pause", paused)
}

// Seek seeks to an absolute position in seconds.
func (m *MpvClient) Seek(seconds float64) error {
	return m.sendCommand("seek", seconds, "absolute")
}

// SetVolume sets the volume (0-100).
func (m *MpvClient) SetVolume(vol float64) error {
	log.Printf("mpv: set volume %.1f", vol)
	return m.setProperty("volume", vol)
}

// SetMute sets the mute state.
func (m *MpvClient) SetMute(muted bool) error {
	return m.setProperty("mute", muted)
}

// ── Internal IPC ─────────────────────────────────────────────────────────────

func (m *MpvClient) sendCommand(args ...interface{}) error {
	id := m.reqID.Add(1)
	msg := map[string]interface{}{
		"command":    args,
		"request_id": id,
	}
	return m.writeJSON(msg)
}

func (m *MpvClient) setProperty(name string, value interface{}) error {
	return m.sendCommand("set_property", name, value)
}

func (m *MpvClient) observeProperty(id int, name string) error {
	return m.sendCommand("observe_property", id, name)
}

func (m *MpvClient) writeJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn == nil {
		return fmt.Errorf("mpv not connected")
	}
	_, err = m.conn.Write(data)
	return err
}

// readLoop reads JSON messages from the mpv IPC socket and dispatches events.
func (m *MpvClient) readLoop() {
	scanner := bufio.NewScanner(m.conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg map[string]json.RawMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		// Check if this is a response to a request.
		if reqIDRaw, ok := msg["request_id"]; ok {
			var reqID int64
			json.Unmarshal(reqIDRaw, &reqID)
			m.pendingMu.Lock()
			if ch, found := m.pending[reqID]; found {
				if data, ok := msg["data"]; ok {
					ch <- data
				}
				close(ch)
				delete(m.pending, reqID)
			}
			m.pendingMu.Unlock()
		}

		// Check for events.
		if eventRaw, ok := msg["event"]; ok {
			var event string
			json.Unmarshal(eventRaw, &event)
			m.handleEvent(event, msg)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("mpv read error: %v", err)
	}
}

func (m *MpvClient) handleEvent(event string, msg map[string]json.RawMessage) {
	switch event {
	case "property-change":
		var name string
		if raw, ok := msg["name"]; ok {
			json.Unmarshal(raw, &name)
		}
		switch name {
		case "time-pos":
			if m.OnTimePos != nil {
				if raw, ok := msg["data"]; ok {
					var pos float64
					if json.Unmarshal(raw, &pos) == nil {
						m.OnTimePos(pos)
					}
				}
			}
		case "duration":
			if m.OnDuration != nil {
				if raw, ok := msg["data"]; ok {
					var dur float64
					if json.Unmarshal(raw, &dur) == nil {
						m.OnDuration(dur)
					}
				}
			}
		}

	case "end-file":
		if m.OnTrackEnd != nil {
			reason := "eof"
			if raw, ok := msg["reason"]; ok {
				json.Unmarshal(raw, &reason)
			}
			var fileError string
			if raw, ok := msg["file_error"]; ok {
				json.Unmarshal(raw, &fileError)
			}
			m.OnTrackEnd(reason, fileError)
		}
	}
}

func (m *MpvClient) waitForSocket(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(m.socketPath); err == nil {
			// Socket file exists, try connecting.
			conn, err := net.DialTimeout("unix", m.socketPath, 500*time.Millisecond)
			if err == nil {
				conn.Close()
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", m.socketPath)
}
