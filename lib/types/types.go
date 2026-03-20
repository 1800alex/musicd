package types

// Track mirrors the server's Track type.
type Track struct {
	ID                 string  `json:"id"`
	Filename           string  `json:"filename"`
	Title              string  `json:"title"`
	Artist             string  `json:"artist"`
	Album              string  `json:"album"`
	Year               int     `json:"year"`
	FilePath           string  `json:"file_path"`
	CoverArtID         string  `json:"cover_art_id"`
	Duration           string  `json:"duration"`
	DurationSec        float64 `json:"duration_seconds"`
	PlaylistPositionID string  `json:"playlist_position_id"` // Unique ID per playlist position (empty for non-playlist contexts)
}
