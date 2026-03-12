# musicd - Multi-Platform Music Server & Player

A comprehensive music server and player system with multiple frontends (web, desktop, iOS, Android) and a standalone music player daemon. Built with Go backend, Nuxt frontend, PostgreSQL database, and support for streaming.

## Features

### Core Music Management
- 🎵 **Audio Format Support**: MP3, OGG, M4A, FLAC
- 🏷️ **ID3 Tag Reading**: Automatically extracts title, artist, album, year, and cover art
- 📚 **Music Library Scanning**: Automatic and manual library scanning and indexing
- 🗄️ **Database Storage**: PostgreSQL-backed persistent storage for metadata
- 📋 **Playlist Management**: Create, edit, and manage playlists with real-time updates
- 🔍 **Advanced Search**: Fast filtering by title, artist, album, genre, year, and more
- 🖼️ **Album Artwork**: Display and manage album cover art from ID3 tags

### Frontend Platforms
- 🌐 **Web Interface**: Modern responsive web app (Nuxt + Vue.js)
- 🖥️ **Desktop App**: Electron-based standalone desktop application
- 📱 **iOS App**: Native iOS app using Capacitor
- 🤖 **Android App**: Native Android app using Capacitor
- 🎨 **Responsive Design**: Works seamlessly across all screen sizes

### Playback & Control
- 🎛️ **Audio Controls**: Play, pause, next, previous, shuffle, repeat, volume control
- 🔊 **Streaming Support**: HTTP streaming with configurable bitrate
- 📡 **RTP Streaming**: Network audio streaming to compatible devices
- 🎙️ **Media Session API**: Integration with OS media controls
- ⌨️ **Keyboard Shortcuts**: Global keyboard shortcuts for playback control
- 🎚️ **Queue Management**: Full queue/playlist queue with drag-and-drop support

### Backend Services
- **musicd**: REST API server for music library, metadata, and playlist management
- **musicplayerd**: Standalone daemon for local music playback with network control
- **Music API**: RESTful API for all music operations
- 🐳 **Docker Support**: Full Docker and Docker Compose configuration

## Prerequisites

- **Docker** and **Docker Compose** (for running services)
- **Git** (for cloning the repository)
- **Make** (for build commands)
- Internet connection to download Docker images

## Installation

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd musicd
   ```

2. **Configure environment** (optional):
   ```bash
   cp docker-compose.yml.example docker-compose.yml
   # Edit docker-compose.yml to set your music directory
   ```

3. **Start the services**:
   ```bash
   make docker-build
   make docker-up
   ```

4. **Access the applications**:
   - Web UI: http://localhost:8080
   - API: http://localhost:8080/api

The complete stack (musicd API server, musicplayerd daemon, and PostgreSQL database) will be running in Docker containers.

## Running the Application

All services run in Docker containers orchestrated by `docker-compose.yml`. An example configuration is provided in `docker-compose.yml.example` with comments for customization.

### Start Services
```bash
make docker-up
```

The application will be available at:
- Web UI: http://localhost:8080
- API: http://localhost:8080/api
- Database: localhost:5432

### Stop Services
```bash
make docker-down
```

### View Logs
```bash
make docker-logs
```

### Development Notes

For frontend development (hot reload), you can run the Nuxt dev server locally:
```bash
cd frontend
corepack yarn install
corepack yarn dev
```

This will start the frontend on http://localhost:3000 (proxying API calls to the backend).

## Configuration

All configuration is managed through `docker-compose.yml` and environment variables.

### Key Configuration

**docker-compose.yml** contains:
- Music directory mount path
- PostgreSQL database credentials
- Service ports
- Environment variables for both services

**Database**: PostgreSQL is automatically initialized with `init.sql` on first run, creating tables for:
- Tracks (music files with metadata)
- Albums
- Artists
- Playlists
- Queue state
- User sessions

### Customization

Edit `docker-compose.yml` to:
- Change music library path (`/home/alex/Music/Playlists` volume)
- Modify service ports (default: 8080 for web UI, 5432 for database)
- Configure streaming settings for musicplayerd
- Adjust environment variables for both services

## API Endpoints

### Web Interface
- `GET /` - Main application page (serves frontend)
- `GET /static/*` - Static assets
- `GET /icons/*` - Application icons

### Music API
- `GET /api/tracks` - Get all tracks with pagination and filtering
- `GET /api/tracks/{id}` - Get track details
- `GET /api/albums` - Get all albums
- `GET /api/artists` - Get all artists
- `GET /api/library/scan` - Scan music library (async)
- `GET /music/:path` - Stream audio file

### Playlists API
- `GET /api/playlists` - Get all playlists
- `GET /api/playlists/{id}` - Get playlist details
- `POST /api/playlists` - Create new playlist
- `PUT /api/playlists/{id}` - Update playlist
- `DELETE /api/playlists/{id}` - Delete playlist
- `POST /api/playlists/{id}/tracks` - Add track to playlist
- `DELETE /api/playlists/{id}/tracks/{trackId}` - Remove track from playlist

### Player Control API (musicplayerd)
- `GET /api/player/status` - Get current playback status
- `POST /api/player/play` - Start playback
- `POST /api/player/pause` - Pause playback
- `POST /api/player/next` - Skip to next track
- `POST /api/player/previous` - Go to previous track
- `POST /api/player/queue/add` - Add tracks to queue
- `GET /api/player/queue` - Get current queue

## Usage

### Web Interface

1. **Browse Your Library**: Access http://localhost:8080 to view your music collection
   - Browse by Artists, Albums, or All Tracks
   - View cover art and metadata
   - Advanced filtering by genre, year, and more

2. **Search**: Use the search bar to find tracks by:
   - Track title
   - Artist name
   - Album name
   - Any metadata field

3. **Create & Manage Playlists**:
   - Create new playlists from the interface
   - Add/remove tracks by dragging or using context menu
   - Playlists are saved to the database

4. **Playback Controls**:
   - Play, pause, next, previous buttons
   - Volume slider
   - Shuffle and repeat modes
   - Queue management

### Desktop & Mobile Apps

- **Electron Desktop**: Full-featured desktop application with offline support
- **iOS App**: Native app with Media Session integration
- **Android App**: Native app with Material Design
- **Web PWA**: Progressive Web App for mobile browsers

### Music Player Daemon (musicplayerd)

The standalone player daemon allows:
- Headless music playback on servers/headless systems
- Remote control via API
- Audio streaming to network devices
- Queue management across network sessions

Control the daemon:
```bash
# Programmatically via API
curl http://localhost:8080/api/player/play
```

### Library Scanning

Automatic library scanning happens on startup. Manual rescan:
- Via web interface: Library menu → Rescan
- Via API: `GET /api/library/scan`
- Extracts metadata from ID3 tags
- Indexes new files and removals

## Technical Details

### Backend (Go)
- **API Framework**: Standard library with modern REST architecture
- **Database**: PostgreSQL for persistent storage with full-text search
- **Music Processing**:
  - dhowden/tag for ID3 metadata extraction
  - File system scanning with recursive indexing
- **Streaming**: FFmpeg/mpv integration for audio encoding and playback
- **Concurrency**: goroutines for parallel library scanning

### Frontend (Nuxt 3)
- **Framework**: Vue 3 with Nuxt 3 for SSR/hybrid rendering
- **UI Components**: Custom component library with Tailwind CSS
- **State Management**: Pinia for centralized state
- **Client Library**: Axios for API communication
- **Media Controls**: Media Session API integration
- **PWA**: Progressive Web App with offline support and install capability

### Multi-Platform Support
- **Web**: Standard browser (Chrome, Firefox, Safari, Edge)
- **Desktop**: Electron with native integrations
- **iOS**: Capacitor bridge to native APIs
- **Android**: Capacitor bridge to native APIs

### Audio Playback
- **musicplayerd**: Built on mpv for reliable playback
- **Supported Formats**: MP3, OGG, FLAC, M4A
- **Streaming**: HTTP streaming, RTP multicast support
- **Quality**: Configurable bitrate, format detection

## Troubleshooting

### Services Not Starting

Check Docker status:
```bash
docker-compose ps
make docker-logs
```

Verify Docker is installed and running:
```bash
docker --version
docker-compose --version
```

### No Music Found

1. Verify music directory path in `docker-compose.yml`
2. Ensure music files are in supported formats: `.mp3`, `.ogg`, `.m4a`, `.flac`
3. Rescan library: Web UI → Library → Rescan or API `GET /api/library/scan`
4. Check logs: `make docker-logs | grep musicd`

### Database Issues

Check database logs:
```bash
docker-compose logs db
```

Reset database (loses all data):
```bash
make db-reset
```

### Web UI Not Loading

1. Verify services are running: `docker-compose ps`
2. Check if port 8080 is available: `lsof -i :8080`
3. Clear browser cache: Ctrl+Shift+Delete (or Cmd+Shift+Delete on macOS)
4. Check browser console (F12) for errors
5. View application logs: `make docker-logs`

### API Endpoint Issues

Test API connectivity:
```bash
curl http://localhost:8080/api/tracks
```

Check application logs for errors:
```bash
make docker-logs
```

## Development

### Docker Development

All development uses Docker containers:

**Build and start services**:
```bash
make docker-build
make docker-up
```

**View container logs**:
```bash
make docker-logs
```

**Stop services**:
```bash
make docker-down
```

### Frontend Development

For frontend hot reload during development:

```bash
make frontend-dev
```

This starts the Nuxt dev server on http://localhost:3000 with API proxying to the backend.

**End-to-end tests** (Playwright):
```bash
make test-e2e
```

### Building Native Apps

**iOS** (requires macOS and Xcode):
```bash
make cap-run-ios
```

**Android** (requires Android SDK):
```bash
make cap-run-android
```

**Electron Desktop**:
```bash
make electron-build
```

### Extending the Application

#### Adding Music Metadata
1. Extend database schema in `init.sql`
2. Rebuild Docker image: `make docker-build`
3. Restart services: `make docker-down && make docker-up`

#### Custom Audio Streaming
1. Modify streaming logic in `cmd/musicplayerd/stream.go`
2. Configure streaming environment variables in `docker-compose.yml`
3. Rebuild and restart: `make docker-build && make docker-up`

#### Adding UI Features
1. Create Vue components in `frontend/app/components/`
2. Use Pinia stores for state management
3. Test with Playwright: `cd frontend && corepack yarn test:e2e`
4. Rebuild Docker image: `make docker-build`

#### Building Native Apps
**iOS** (requires macOS and Xcode):
```bash
make cap-run-ios
```

**Android** (requires Android SDK):
```bash
make cap-run-android
```

**Electron Desktop** (cross-platform):
```bash
make electron-build
```

## Deployment

### Docker Deployment

Deploy the complete stack (API server, player daemon, database) with:

```bash
make docker-build
make docker-up
```

Services will be available at:
- Web UI: http://localhost:8080
- API: http://localhost:8080/api
- Database: localhost:5432

### Production Deployment

For production deployments:

1. **Use a reverse proxy** (nginx, Caddy) for HTTPS and load balancing
2. **Set up environment variables** in docker-compose.yml for:
   - Database credentials
   - Music library paths
   - Service configurations
3. **Configure persistent volumes** for:
   - Database data (`postgres_data`)
   - Player daemon state (`musicplayerd_data`)
   - Music library
4. **Regular backups** of PostgreSQL data
5. **Monitoring and logging** for all services

Example production docker-compose configuration:
```yaml
services:
  app:
    image: musicd:latest
    restart: unless-stopped
    environment:
      DATABASE_URL: postgres://user:pass@db:5432/musicdb
    volumes:
      - /path/to/music:/music
  musicplayerd:
    image: musicd:latest
    command: ./bin/musicplayerd
    depends_on:
      - app
  db:
    image: postgres:15-alpine
    restart: unless-stopped
    volumes:
      - postgres_data:/var/lib/postgresql/data
volumes:
  postgres_data:
```

## Architecture

```
┌────────────────────────────────────────┐
│   Multi-Platform Frontend              │
├──────┬─────────┬────────┬──────────────┤
│ Web  │Electron │ iOS    │  Android     │
└──────┴─────────┴────────┴──────────────┘
         │
    REST API
         │
┌────────────────────────────────────┐
│        musicd (API Server)         │
├────────────────────────────────────┤
│  Routes │ Handler │ Database Logic │
└────────────────────────────────────┘
         │
    PostgreSQL
         │
┌────────────────────────────────────┐
│   musicplayerd (Player Daemon)     │
├──────────────┬─────────────────────┤
│  mpv Player  │ Queue + Streaming   │
└──────────────┴─────────────────────┘
         │
    Audio Output
```

## Contributing

Contributions are welcome! Areas for enhancement:
- Transcoding support for additional audio formats
- Additional audio codecs and streaming protocols
- Performance optimizations for large libraries
- Additional metadata providers (MusicBrainz, Spotify API integration)
- Lyrics fetching and display (e.g. Genius API)
- Mobile app improvements
- Documentation and examples

## License

This project is provided as-is. Feel free to modify and use for your needs.
