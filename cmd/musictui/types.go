package main

import (
	"encoding/json"
	"musicd/lib/types"
)

// APIResponse is the paginated envelope for list endpoints.
type APIResponse struct {
	Data       json.RawMessage `json:"data"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
	Total      int             `json:"total"`
	Search     string          `json:"search"`
}

// SessionInfo from /api/sessions.
type SessionInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	HasPlayer       bool   `json:"has_player"`
	ControllerCount int    `json:"controller_count"`
}

// PlayerState is the deserialized WebSocket state broadcast.
type PlayerState struct {
	Type            string                 `json:"type"`
	IsPlaying       bool                   `json:"is_playing"`
	CurrentTrack    *types.Track           `json:"current_track"`
	CurrentTime     float64                `json:"current_time"`
	Duration        float64                `json:"duration"`
	Volume          float64                `json:"volume"`
	Muted           bool                   `json:"muted"`
	Shuffle         bool                   `json:"shuffle"`
	RepeatMode      string                 `json:"repeat_mode"`
	Queue           []types.Track          `json:"queue"`
	TemporaryQueue  []types.Track          `json:"temporary_queue"`
	CurrentPlaylist map[string]interface{} `json:"current_playlist"`
}

// Command is sent over WebSocket to the player.
type Command struct {
	Type   string      `json:"type"`
	Action string      `json:"action"`
	Value  interface{} `json:"value,omitempty"`
}
