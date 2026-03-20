# musictui - TUI Remote Control for musicd

## Current Features
- [x] CLI flag `--url` for daemon URL
- [x] URL configurable in TUI via Connect page
- [x] Session picker to select a player session to control
- [x] Browse tracks with pagination and search
- [x] Browse artists with pagination and search
- [x] Browse playlists
- [x] Artist detail page (albums + tracks)
- [x] Album detail page
- [x] Playlist detail page with pagination and search
- [x] Queue view (live from WebSocket state)
- [x] Vim-style navigation (h: back, l/Enter: select, j/k: up/down)
- [x] Pagination with [ and ] keys
- [x] Search with /
- [x] Volume controls (+/-)
- [x] Shuffle toggle (s)
- [x] Repeat mode cycling (r: Off/All/One)
- [x] Play/pause (Space), next (n), previous (N)
- [x] Mute toggle (m)
- [x] Now-playing status bar with progress, volume, shuffle/repeat
- [x] Auto-reconnect WebSocket with exponential backoff
- [x] Help overlay (?)

## Future Enhancements

### Stream music locally via mpv
- Add `--stream` flag to spawn a local mpv process
- Connect to musicd's audio stream endpoint (HTTP/RTP)
- Reuse musicplayerd functionality as a shared package
  - Extract mpv client, queue management, stream manager into `pkg/`
  - Make them importable and unit testable
  - Both musicplayerd and musictui can use the shared packages

### Playlist management
- Create new playlists (POST /api/playlist/create)
- Delete playlists (DELETE /api/playlist/{id})
- Add tracks to playlists (POST /api/playlist/{id}/add/{trackId})
- Remove tracks from playlists (DELETE /api/playlist/{id}/remove/{trackId})

### Cover art display
- Sixel protocol support for terminals that support it
- Auto-detect terminal sixel capability
- Graceful fallback to text-only display
- Side panel on artist/album detail pages

### Lyrics display
- Use /api/lyrics endpoint
- Show lyrics in a scrollable panel

### Mouse support
- Click to select items in tables/lists
- Click on progress bar to seek
- Click on volume to adjust

### Configuration file
- `~/.config/musictui/config.toml`
- Persist server URL, default session, theme preferences
- Auto-connect to last session on startup

### Theme customization
- Dark/light/custom color schemes
- Configurable via config file

### Queue management
- Add tracks to priority queue (queue_add command)
- Clear queue (queue_clear command)
- Reorder queue items
