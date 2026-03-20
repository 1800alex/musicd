package main

import (
	"encoding/json"
	"fmt"
	"io"
	"musicd/lib/types"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// APIClient handles all HTTP REST communication with musicd.
type APIClient struct {
	BaseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new REST API client.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *APIClient) get(path string, params url.Values) ([]byte, error) {
	u := c.BaseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, path)
	}
	return io.ReadAll(resp.Body)
}

func paginationParams(page, pageSize int, search string) url.Values {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.Itoa(pageSize))
	}
	if search != "" {
		params.Set("search", search)
	}
	return params
}

// GetSessions returns all active player sessions.
func (c *APIClient) GetSessions() ([]SessionInfo, error) {
	data, err := c.get("/api/sessions", nil)
	if err != nil {
		return nil, err
	}
	var sessions []SessionInfo
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, fmt.Errorf("decode sessions: %w", err)
	}
	return sessions, nil
}

// GetTracks returns a paginated list of tracks.
func (c *APIClient) GetTracks(page, pageSize int, search string) (*APIResponse, []types.Track, error) {
	data, err := c.get("/api/tracks", paginationParams(page, pageSize, search))
	if err != nil {
		return nil, nil, err
	}
	var resp APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("decode tracks response: %w", err)
	}
	var tracks []types.Track
	if err := json.Unmarshal(resp.Data, &tracks); err != nil {
		return nil, nil, fmt.Errorf("decode tracks data: %w", err)
	}
	return &resp, tracks, nil
}

// GetArtists returns a paginated list of artists.
func (c *APIClient) GetArtists(page, pageSize int, search string) (*APIResponse, []types.Artist, error) {
	data, err := c.get("/api/artists", paginationParams(page, pageSize, search))
	if err != nil {
		return nil, nil, err
	}
	var resp APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("decode artists response: %w", err)
	}
	var artists []types.Artist
	if err := json.Unmarshal(resp.Data, &artists); err != nil {
		return nil, nil, fmt.Errorf("decode artists data: %w", err)
	}
	return &resp, artists, nil
}

// GetArtist returns artist detail with albums and tracks.
func (c *APIClient) GetArtist(id string) (*types.Artist, error) {
	data, err := c.get("/api/artist/"+id, nil)
	if err != nil {
		return nil, err
	}
	var artist types.Artist
	if err := json.Unmarshal(data, &artist); err != nil {
		return nil, fmt.Errorf("decode artist: %w", err)
	}
	return &artist, nil
}

// GetAlbum returns album detail with tracks.
func (c *APIClient) GetAlbum(id string) (*types.Album, error) {
	data, err := c.get("/api/album/"+id, nil)
	if err != nil {
		return nil, err
	}
	var album types.Album
	if err := json.Unmarshal(data, &album); err != nil {
		return nil, fmt.Errorf("decode album: %w", err)
	}
	return &album, nil
}

// GetAlbumTracks returns paginated tracks for an album.
func (c *APIClient) GetAlbumTracks(id string, page, pageSize int, search string) (*APIResponse, []types.Track, error) {
	data, err := c.get("/api/album/"+id+"/tracks", paginationParams(page, pageSize, search))
	if err != nil {
		return nil, nil, err
	}
	var resp APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("decode album tracks response: %w", err)
	}
	var tracks []types.Track
	if err := json.Unmarshal(resp.Data, &tracks); err != nil {
		return nil, nil, fmt.Errorf("decode album tracks data: %w", err)
	}
	return &resp, tracks, nil
}

// GetPlaylists returns all playlists (not paginated).
func (c *APIClient) GetPlaylists() ([]types.Playlist, error) {
	data, err := c.get("/api/playlists", nil)
	if err != nil {
		return nil, err
	}
	var playlists []types.Playlist
	if err := json.Unmarshal(data, &playlists); err != nil {
		return nil, fmt.Errorf("decode playlists: %w", err)
	}
	return playlists, nil
}

// GetPlaylistTracks returns paginated tracks for a playlist.
func (c *APIClient) GetPlaylistTracks(id string, page, pageSize int, search string) (*APIResponse, []types.Track, error) {
	data, err := c.get("/api/playlist/"+id+"/tracks", paginationParams(page, pageSize, search))
	if err != nil {
		return nil, nil, err
	}
	var resp APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("decode playlist tracks response: %w", err)
	}
	var tracks []types.Track
	if err := json.Unmarshal(resp.Data, &tracks); err != nil {
		return nil, nil, fmt.Errorf("decode playlist tracks data: %w", err)
	}
	return &resp, tracks, nil
}
