package main

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSClient manages the WebSocket connection to musicd as a controller.
type WSClient struct {
	baseURL   string
	sessionID string

	conn    *websocket.Conn
	connMu  sync.Mutex
	writeMu sync.Mutex

	onState func(PlayerState)
	onError func(error)

	done   chan struct{}
	closed bool
	mu     sync.Mutex
}

// NewWSClient creates a new WebSocket client for controlling a session.
func NewWSClient(baseURL, sessionID string, onState func(PlayerState), onError func(error)) *WSClient {
	return &WSClient{
		baseURL:   baseURL,
		sessionID: sessionID,
		onState:   onState,
		onError:   onError,
		done:      make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection and starts the read loop.
func (w *WSClient) Connect() error {
	wsURL := w.baseURL
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL = strings.TrimSuffix(wsURL, "/")
	wsURL += "/api/ws/control/" + w.sessionID

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	w.connMu.Lock()
	w.conn = conn
	w.connMu.Unlock()

	go w.readLoop()
	return nil
}

func (w *WSClient) readLoop() {
	for {
		w.connMu.Lock()
		conn := w.conn
		w.connMu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			w.mu.Lock()
			closed := w.closed
			w.mu.Unlock()
			if closed {
				return
			}
			if w.onError != nil {
				w.onError(err)
			}
			w.reconnect()
			return
		}

		var msg map[string]json.RawMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		var msgType string
		if t, ok := msg["type"]; ok {
			json.Unmarshal(t, &msgType)
		}

		if msgType == "state" && w.onState != nil {
			var state PlayerState
			if err := json.Unmarshal(message, &state); err == nil {
				w.onState(state)
			}
		}
	}
}

func (w *WSClient) reconnect() {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		w.mu.Lock()
		closed := w.closed
		w.mu.Unlock()
		if closed {
			return
		}

		time.Sleep(backoff)

		if err := w.Connect(); err != nil {
			log.Printf("Reconnect failed: %v", err)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		return
	}
}

// SendCommand sends a command to the player via WebSocket.
func (w *WSClient) SendCommand(action string, value interface{}) error {
	w.connMu.Lock()
	conn := w.conn
	w.connMu.Unlock()

	if conn == nil {
		return nil
	}

	cmd := Command{
		Type:   "command",
		Action: action,
		Value:  value,
	}

	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	return conn.WriteJSON(cmd)
}

// Close shuts down the WebSocket connection.
func (w *WSClient) Close() {
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()

	w.connMu.Lock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.connMu.Unlock()
}

// IsConnected returns whether the WebSocket is currently connected.
func (w *WSClient) IsConnected() bool {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	return w.conn != nil
}
