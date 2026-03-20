package types

import "time"

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
	PlaylistPositionID string `json:"playlist_position_id"` // Unique ID per playlist position (empty for non-playlist contexts)
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

type SessionInfo struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	HasPlayer       bool      `json:"has_player"`
	ControllerCount int       `json:"controller_count"`
	LastSeen        time.Time `json:"last_seen"`
}
