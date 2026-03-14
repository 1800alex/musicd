# musicd - Multi-Platform Music Server & Player

> **Disclaimer**: This project was built with AI assistance. The core backend (Go) and lower-level frontend services were mostly written manually, but AI was instrumental in building the UI/visual components and helping structure the overall architecture. The project grew larger than originally anticipated, and turned out to be a much bigger undertaking than expected. I am sharing it in its current state in case it can be useful to others, but please be aware that it is still a work in progress and may have bugs, incomplete features, and areas that need improvement. Contributions are welcome!

A comprehensive music server and player system with multiple frontends (web, desktop, iOS, Android) and a standalone music player daemon. Built with Go backend, Nuxt frontend, PostgreSQL database, and support for streaming.

Perfect for self-hosting your music library and streaming to any device.

## Features

### Core Music Management
- 🎵 **Audio Format Support**: MP3, OGG, M4A, FLAC
- 🏷️ **ID3 Tag Reading**: Automatically extracts title, artist, album, year, and cover art
- 📚 **Music Library Scanning**: Automatic and manual library scanning and indexing
- 🗄️ **Database Storage**: PostgreSQL-backed persistent storage for metadata
- 📋 **Playlist Management**: Create, edit, and manage playlists with real-time updates
- 🔍 **Advanced Search**: Fast filtering by title, artist, album, genre, year, and more
- 🖼️ **Album Artwork**: Display and manage album cover art from ID3 tags

### Access Your Music Anywhere
- 🌐 **Web Interface**: Modern responsive web app in any browser
- 🖥️ **Desktop App**: Standalone Electron app for Linux, macOS, Windows
- 📱 **Mobile Apps**: iOS and Android apps available
  - ⚠️ **iOS**: Currently requires Apple Developer Account for building (help needed!)
  - ⚠️ **Android**: Currently untested (I don't have an android phone, help needed!)
- 🎨 **Responsive Design**: Works seamlessly across all screen sizes

### Playback & Control
- 🎛️ **Audio Controls**: Play, pause, next, previous, shuffle, repeat, volume control
- 🔊 **HTTP Streaming**: Stream audio to any connected device
- 📡 **RTP Streaming**: Network audio streaming to compatible devices
- 🎙️ **Media Session API**: Integration with OS media controls (desktop/mobile)
- ⌨️ **Keyboard Shortcuts**: Global keyboard shortcuts for playback control
- 🎚️ **Queue Management**: Add tracks/albums/playlists to queue
- 🌐 **Remote Session Control**: Control playback from any device with API access

## Why I Built This

I created musicd to solve specific needs that existing music servers didn't address:

1. **Complete Offline Support**: I wanted a music server that works completely offline on my local network. No cloud dependency, no account required. Your music library stays with you.

2. **Cross-Device Queue Control**: I wanted to control the currently playing track and queue from any device—phone, laptop, or desktop. Start playing on one device, switch to another, and control playback of the original session seamlessly. (I use this out in my garage with my phone while the music is actually playing through my garage PC connected to my sound system.)

3. **Support for Existing M3U Playlists**: I've manually curated M3U playlists over the years, and they're precious to me. I needed a solution that respects and works with these existing playlist files instead of forcing me to recreate them.

4. **Playlist Synchronization**: I sync my M3U files across multiple devices already. I wanted musicd to integrate with this workflow, allowing me to edit playlists in one place and have them automatically available everywhere.

The result is a self-hosted music system that puts you in control, respects your existing music organization, and works seamlessly across all your devices.

## What This Project Doesn't Do

It's important to understand what musicd is *not* designed for:

- **Music Streaming Service**: This is not a Spotify/Apple Music replacement. It plays music from *your local files*, not streaming services.

- **User Authentication & Multi-User Support**: musicd doesn't have built-in user accounts or permission management. It's designed for personal or small household use on a trusted local network. I'm open to suggestions for how to implement basic authentication without overcomplicating the setup, but for now it's a single-user system.

- **Automatic Metadata Fetching**: musicd relies on ID3 tags in your music files. It doesn't automatically fetch metadata from MusicBrainz, Spotify, or other online services. Your metadata is only as good as your tags. This is a future enhancement I may add, but for now it's a local library manager that expects you to have properly tagged files.

- **Audio Transcoding**: All audio is streamed in its original format. There's no on-the-fly transcoding to reduce bandwidth or storage usage.

- **Music Discovery**: This isn't a recommendation engine. It's a library manager and player for music you already have. If you want features like music discovery, personalized recommendations, or integration with streaming services, I'm open to ideas on how we could add this via another container or plugin system, but it's not in the core scope of musicd currently.

## Current Limitations & Missing Features

Here's what's **not yet implemented**:

- **Tag Editing**: You cannot edit ID3 tags through musicd. All metadata comes from your files. Make sure your music is properly tagged before importing.
- **Cover Art Upload**: You can view cover art extracted from ID3 tags, but cannot upload or change cover art through the UI.
- **Playlist Metadata Editing**: You can add/remove tracks from playlists (and changes sync to M3U files), but cannot rename playlists or edit playlist descriptions through the UI.
- **Remote Playlist Editing**: While cross-device queue control works perfectly, playlist editing must be done on the device running the player daemon.

## Quick Start with Docker

### Prerequisites

- **Docker** and **Docker Compose** installed
- **Git** for cloning the repository
- Music files in a directory on your system

## Installation

1. **Clone the example docker-compose file**:

Setup your docker-compose file by copying the example and configuring it:
```bash
wget https://raw.githubusercontent.com/1800alex/musicd/main/examples/prod/docker-compose.yml -O docker-compose.yml
```

2. **Configure environment** (optional):
Edit `docker-compose.yml` to set:
- Music library path
- Database credentials
- Service ports

3. **Start the services**:
```bash
docker-compose pull
docker-compose up -d
```

4. **Access the applications**:
   - **Web UI**: http://localhost:8080
   - **API**: http://localhost:8080/api

The first startup will:
- Initialize the PostgreSQL database
- Scan your music library
- Index all metadata

## Running the Desktop App

Download the latest release from [GitHub Releases](https://github.com/1800alex/musicd/releases):

- **Linux**: `Music.Player-*.AppImage`
- **macOS**: `Music.Player-*.dmg`
- **Windows**: `Music.Player-*.exe`

The desktop app connects to your running musicd server (local or remote).

### Running the AppImage on Linux

For Linux:
```bash
chmod a+x Music.Player-0.0.1.AppImage
./Music.Player-0.0.1.AppImage

# if you get an error about sandbox issues, try:
./Music.Player-0.0.1.AppImage --no-sandbox
```

## Running Just the Player Daemon

For headless systems or minimal resource usage, run just the music player daemon:

1. **Clone the example docker-compose file**:

Setup your docker-compose file by copying the example and configuring it:
```bash
wget https://raw.githubusercontent.com/1800alex/musicd/main/examples/musicplayerd-only/docker-compose.yml -O docker-compose.yml
```

2. **Configure environment** (optional):
Edit `docker-compose.yml` to set:
- IP address of the musicd API server
- Session name for the player daemon

3. **Start the services**:
```bash
docker-compose pull
docker-compose up -d
```

Now you should be able to control playback via the API or web interface while keeping the player daemon running on a separate machine or in a lightweight container.

## Usage

### Web Interface

1. **Browse Your Library**: Visit http://localhost:8080
   - Browse by Artists, Albums, Playlists, or All Tracks
   - View cover art and metadata
   - Advanced filtering by genre, year, and more

2. **Search**: Use the search bar to find tracks by:
   - Track title
   - Artist name
   - Album name

3. **Create & Manage Playlists**:
   - Create new playlists from the interface
   - Add/remove tracks by dragging or using context menu
   - Playlists are saved persistently both in the database and as M3U files

4. **Playback Controls**:
   - Play, pause, next, previous buttons
   - Volume slider
   - Shuffle and repeat modes
   - Queue management

### Library Scanning

Automatic scanning happens on startup in addition to a file watcher. Manual rescan is available as well:
- Via web interface: Menu → Library Scan
- Via API: `GET /api/library/scan`

## Raspberry Pi / aarch64 Support

Docker images are built for both `amd64` and `aarch64` architectures, making musicd perfect for Raspberry Pi:

```bash
# Docker automatically pulls the correct image for your architecture
docker-compose up -d
```

Supported on:
- Raspberry Pi 3B+ and newer
- Other aarch64 Linux systems

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
         │                    │
    PostgreSQL                │
                              │
┌────────────────────────────────────┐
│   musicplayerd (Player Daemon)     │
├──────────────┬─────────────────────┤
│  mpv Player  │ Queue + Streaming   │
└──────────────┴─────────────────────┘
         │
    Audio Output
```

## API Endpoints

### Music Library
- `GET /api/tracks` - Get all tracks with pagination and filtering
- `GET /api/tracks/{id}` - Get track details
- `GET /api/albums` - Get all albums
- `GET /api/artists` - Get all artists
- `GET /api/library/scan` - Scan music library (async)

### Streaming
- `GET /music/:path` - Stream audio file

### Playlists
- `GET /api/playlists` - Get all playlists
- `POST /api/playlists` - Create new playlist
- `PUT /api/playlists/{id}` - Update playlist
- `DELETE /api/playlists/{id}` - Delete playlist
- `POST /api/playlists/{id}/tracks` - Add track to playlist
- `DELETE /api/playlists/{id}/tracks/{trackId}` - Remove track from playlist

### Player Control
- `GET /api/player/status` - Get current playback status
- `POST /api/player/play` - Start playback
- `POST /api/player/pause` - Pause playback
- `POST /api/player/next` - Skip to next track
- `POST /api/player/previous` - Go to previous track

## Troubleshooting

### Services Not Starting

Check Docker status:
```bash
docker-compose ps
docker-compose logs
```

Verify Docker is installed:
```bash
docker --version
docker-compose --version
```

### No Music Found

1. Verify music directory path in `docker-compose.yml`
2. Ensure music files are in supported formats: `.mp3`, `.ogg`, `.m4a`, `.flac`
3. Rescan library: Web UI → Menu → Library Scan
4. Check logs: `docker-compose logs app`

### Web UI Not Loading

1. Verify services are running: `docker-compose ps`
2. Check if port 8080 is available: `lsof -i :8080`
3. Clear browser cache: Ctrl+Shift+Delete (Cmd+Shift+Delete on macOS)
4. Check browser console (F12) for errors

### Database Issues

Check database logs:
```bash
docker-compose logs db
```

Reset database (loses all data):
```bash
docker-compose down -v
docker-compose up -d
```

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for:
- Building from source
- Running development servers
- Building native apps (iOS, Android, Electron)
- Contributing guidelines
- Architecture details

## Deployment

### Docker Deployment

For production deployments:

1. **Use a reverse proxy** (nginx, Caddy) for HTTPS and load balancing
2. **Set secure environment variables** for database credentials
3. **Configure persistent volumes** for:
   - Database data
   - Music library
4. **Enable regular backups** of PostgreSQL data
5. **Set up monitoring and logging** for all services

## Contributing

Contributions are welcome! Areas for enhancement:
- User authentication and multi-user support is completely unimplemented (help needed!)
- Adding new playlists does not give users the option to specify the file path for the generated M3U playlist (help needed!)
- iOS code signing and testing (need Apple Developer Account)
- Android build testing and improvements
- Transcoding support for additional audio formats
- Currently all audio is streamed as-is without transcoding. Adding on-the-fly transcoding would be slow, but if we pre-transcode and store additional formats we risk using more disk space. A possible solution is to add a transcoding queue that generates additional formats in the background after scanning new tracks.
- Add support for metadata providers (MusicBrainz, Spotify API integration), currently all metadata is read from ID3 tags which can be incomplete or incorrect. Fetching metadata from online sources would allow for better organization and display of the music library.
- Lyrics fetching and display is not currently implemented (help needed!)
- Documentation and examples

### Build Status

The project uses GitHub Actions for automated builds:
- **On Push Tags** (`v*`): Builds and releases all platforms
  - Docker images for amd64 and aarch64
  - Desktop apps for Linux, macOS, Windows
  - Release with artifacts

## License

This project is licensed under the GPL-3.0 License - see the [COPYING](COPYING) file for details.
