package main

import "encoding/json"

// Track mirrors the server Track type.
type Track struct {
	ID                 string `json:"id"`
	Filename           string `json:"filename"`
	Title              string `json:"title"`
	Artist             string `json:"artist"`
	Album              string `json:"album"`
	Year               int    `json:"year"`
	FilePath           string `json:"file_path"`
	FileHash           string `json:"file_hash"`
	CoverArtID         string `json:"cover_art_id"`
	Duration           string `json:"duration"`
	DurationSec        int    `json:"duration_seconds"`
	PlaylistPositionID string `json:"playlist_position_id"`
}

// Artist from list endpoint.
type Artist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CoverArtID string `json:"cover_art_id"`
	TrackCount int    `json:"track_count"`
}

// ArtistDetail from detail endpoint with nested albums/tracks.
type ArtistDetail struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	CoverArtID string   `json:"cover_art_id"`
	Albums     []*Album `json:"albums"`
	Tracks     []Track  `json:"tracks"`
	TrackCount int      `json:"track_count"`
}

// Album represents an album with its tracks.
type Album struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Artist     string  `json:"artist"`
	Year       int     `json:"year"`
	Tracks     []Track `json:"tracks"`
	TrackCount int     `json:"track_count"`
	CoverArtID string  `json:"cover_art_id"`
}

// Playlist from list endpoint.
type Playlist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	TrackCount  int    `json:"track_count"`
	CoverArtID  string `json:"cover_art_id"`
}

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
	CurrentTrack    *Track                 `json:"current_track"`
	CurrentTime     float64                `json:"current_time"`
	Duration        float64                `json:"duration"`
	Volume          float64                `json:"volume"`
	Muted           bool                   `json:"muted"`
	Shuffle         bool                   `json:"shuffle"`
	RepeatMode      string                 `json:"repeat_mode"`
	Queue           []Track                `json:"queue"`
	TemporaryQueue  []Track                `json:"temporary_queue"`
	CurrentPlaylist map[string]interface{} `json:"current_playlist"`
}

// Command is sent over WebSocket to the player.
type Command struct {
	Type   string      `json:"type"`
	Action string      `json:"action"`
	Value  interface{} `json:"value,omitempty"`
}
