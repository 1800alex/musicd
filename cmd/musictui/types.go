package main

import (
	"musicd/lib/types"
)

// PlayerState is the deserialized WebSocket state broadcast.
type PlayerState struct {
	Type                 string                 `json:"type"`
	IsPlaying            bool                   `json:"is_playing"`
	CurrentTrack         *types.Track           `json:"current_track"`
	CurrentTime          float64                `json:"current_time"`
	Duration             float64                `json:"duration"`
	Volume               float64                `json:"volume"`
	Muted                bool                   `json:"muted"`
	Shuffle              bool                   `json:"shuffle"`
	RepeatMode           string                 `json:"repeat_mode"`
	QueueLength          int                    `json:"queue_length"`
	TemporaryQueueLength int                    `json:"temporary_queue_length"`
	CurrentPlaylist      map[string]interface{} `json:"current_playlist"`
}

// Command is sent over WebSocket to the player.
type Command struct {
	Type   string      `json:"type"`
	Action string      `json:"action"`
	Value  interface{} `json:"value,omitempty"`
}
