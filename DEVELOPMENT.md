# musicd - Development Guide

This guide covers building, testing, and developing musicd from source.

## Prerequisites

- **Go** 1.21+ (for backend development)
- **Node.js** 24+ with Yarn (for frontend development)
- **Docker** and **Docker Compose** (for database and services)
- **Make** (for build commands)
- **PostgreSQL** (for local database development, or use Docker)

## Project Structure

```
musicd/
├── cmd/
│   ├── musicd/          # Main API server
│   ├── musicplayerd/    # Standalone player daemon
│   └── icongen/         # Icon generation utility
├── frontend/            # Nuxt 3 frontend (Vue.js)
├── init.sql            # Database schema
├── Makefile            # Build commands
└── docker-compose.yml  # Local development setup
```

## Quick Start

### Start Development Environment

```bash
# Start database in Docker
make db-up

# Build and start API server
make run

# In another terminal, start frontend dev server
make frontend-dev
```

Access the application at http://localhost:3000/ui/

### Stop Development Environment

```bash
make docker-down
```

## Development Tasks

### Backend Development (Go)

**Build the application**:
```bash
make build
```

**Run locally**:
```bash
make run
```

**Format code**:
```bash
make fmt
```

**Run tests**:
```bash
make test
```

### Frontend Development (Nuxt 3)

**Install dependencies**:
```bash
cd frontend
corepack yarn install
```

**Start dev server with hot reload**:
```bash
make frontend-dev
# or
cd frontend && corepack yarn dev
```

The dev server runs at http://localhost:3000 and proxies API calls to the backend.

**Run end-to-end tests**:
```bash
make test-e2e
```

**Run tests with browser visible**:
```bash
make test-e2e-headed
```

**View test report**:
```bash
make test-e2e-report
```

**Format code**:
```bash
corepack yarn prettier
```

### Docker Development

**Build Docker images**:
```bash
make docker-build
```

**Start all services**:
```bash
make docker-up
```

**View logs**:
```bash
make docker-logs
```

**Stop services**:
```bash
make docker-down
```

**Reset database** (loses all data):
```bash
make db-reset
```

## Building Native Apps

### Capacitor Setup (iOS & Android)

Capacitor is used to wrap the web app as native iOS and Android apps.

**Sync web assets to native projects**:
```bash
make cap-sync
```

**Android only**:
```bash
make cap-sync-android
```

**iOS only**:
```bash
make cap-sync-ios
```

### iOS Build

**Requirements**:
- macOS with Xcode installed
- Apple Developer Account (required for code signing)

**Open iOS project in Xcode**:
```bash
make cap-open-ios
```

**Build and run on simulator**:
```bash
make cap-run-ios
```

**Current Status**: iOS builds require an Apple Developer Account for code signing. Contributors with a developer account are welcome to help set up proper code signing.

### Android Build

**Requirements**:
- Android SDK and Android Studio
- Java 21+

**Open Android project in Android Studio**:
```bash
make cap-open-android
```

**Build and run on emulator/device**:
```bash
make cap-run-android
```

**Current Status**: Android build is currently untested. Contributors are welcome to help test and report issues.

### Electron Desktop Build

**Build for all platforms**:
```bash
make electron-build
```

This creates installers for:
- Linux: AppImage
- macOS: DMG
- Windows: NSIS installer + portable EXE

**Development mode**:
```bash
make electron-dev
```

Starts Electron with hot reload for frontend changes.

## Icon Generation

Generate app icons and splash screens:
```bash
make icon
```

This generates icons in multiple sizes for:
- Web app
- PWA
- iOS
- Android
- Favicon

## Database

### Schema

The database schema is defined in `init.sql` and automatically initializes on first run. It includes tables for:
- Tracks (music files with metadata)
- Albums
- Artists
- Playlists
- Queue state
- User sessions

### Local Development

**Start PostgreSQL in Docker**:
```bash
make db-up
```

**Stop database**:
```bash
make db-down
```

**Reset database** (clears all data):
```bash
make db-reset
```

### Connection Details

When running with Docker:
- Host: `localhost`
- Port: `5432`
- Database: `musicdb`
- User: `musicuser`
- Password: Set in `docker-compose.yml`

## Configuration

### Environment Variables

Create a `.env.development` file in the `frontend/` directory for frontend configuration:
```bash
# Backend URL for dev server
NUXT_PUBLIC_BACKEND_URL=http://localhost:8080
```

### Docker Compose Configuration

The `docker-compose.yml` file defines:
- Music library path
- Database credentials
- Service ports
- Environment variables

Edit this file to customize the development environment.

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run specific test
go test ./cmd/musicd/... -v
```

### End-to-End Tests

```bash
# Run in headless mode
make test-e2e

# Run with browser visible
make test-e2e-headed

# View report
make test-e2e-report
```

Tests are located in `frontend/tests/` and use Playwright.

## Code Style

### Go

```bash
make fmt
```

Uses standard Go formatting.

### Frontend (Vue/Nuxt)

```bash
cd frontend && corepack yarn prettier
```

Uses Prettier for consistent code formatting.

## Architecture

### Backend (Go)

- **API Framework**: Standard library with modern REST architecture
- **Database**: PostgreSQL for persistent storage
- **Music Processing**: dhowden/tag for ID3 metadata extraction
- **Streaming**: HTTP streaming support
- **Concurrency**: goroutines for parallel operations

### Frontend (Nuxt 3)

- **Framework**: Vue 3 with Nuxt 3
- **UI Components**: Buefy + Bulma CSS
- **State Management**: Pinia
- **API Client**: Axios
- **Media Controls**: Media Session API
- **PWA**: Progressive Web App support

### Multi-Platform

- **Web**: Standard browser
- **Desktop**: Electron with native integrations
- **iOS**: Capacitor bridge to native APIs
- **Android**: Capacitor bridge to native APIs

## Debugging

### Backend

```bash
# Run with verbose output
go run ./cmd/musicd -v

# View database logs
docker logs musicd-db-1
```

### Frontend

```bash
# Browser dev tools (F12)
# Check console for errors
# Use Vue DevTools extension
```

### Docker

```bash
# View service logs
make docker-logs

# View specific service
docker-compose logs musicd
```

## Common Issues

### Port Already in Use

```bash
# Find process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>
```

### Database Connection Issues

```bash
# Check database status
docker-compose ps db

# View database logs
docker-compose logs db

# Reset database
make db-reset
```

### Frontend Build Errors

```bash
# Clear build cache
cd frontend && rm -rf .nuxt .output

# Reinstall dependencies
corepack yarn install --force

# Try building again
make frontend
```

### Git LFS Issues

If you encounter issues with large files:
```bash
# Install git-lfs
git lfs install

# Pull all LFS files
git lfs pull
```

## Android

First make sure you have `javac` (Java 21+) installed and available in your PATH. For me on ubuntu 25 I did: `sudo apt install openjdk-25-jdk`

Download and set up [Android Studio](https://developer.android.com/studio/install) with the Android SDK.

Unpack the Android Studio distribution archive that you downloaded where you wish to install the program.

To start the application, open a console, cd into "{installation home}/bin" and type: `./studio` this will download and install the sdk and emulator.

```bash
ANDROID_SDK_ROOT=~/Android/Sdk make cap-run-android
```


## Version Management

The project uses semantic versioning for releases. Version tags should follow the `v*` pattern (e.g., `v1.0.0`, `v0.1.0`).

GitHub Actions workflows automatically build and release when tags are pushed:
- Docker images pushed to GitHub Container Registry
- Desktop apps built for Linux, macOS, Windows
- Release created with artifacts

## Contributing

When contributing:

1. **Fork and create a feature branch**
2. **Follow code style** (run `make fmt` for Go, `corepack yarn prettier` for frontend)
3. **Write tests** for new functionality
4. **Test on multiple platforms** (at minimum test backend and web UI)
5. **Submit a pull request** with description of changes

### Building on CI/CD

The project uses GitHub Actions for automated builds:
- **On Push Tags** (`v*`): Build all platforms and create release
- **All Workflows**: Use Git LFS for large files
