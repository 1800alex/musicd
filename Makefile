.PHONY: docker-build docker-up docker-down

# Default target
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build        - Build the Go application"
	@echo "  run          - Run the application locally"
	@echo "  dev          - Run in development mode (with auto-reload)"
	@echo "  frontend     - Build the frontend assets"
	@echo "  frontend-dev - Run the frontend in development mode"
	@echo "  fmt          - Format code"
	@echo "  test         - Run Go tests"
	@echo "  test-e2e     - Run frontend Playwright e2e tests (headless)"
	@echo "  test-e2e-headed - Run e2e tests with browser visible"
	@echo "  test-e2e-report - Open HTML report from last e2e run"
	@echo "  clean        - Clean build artifacts"
	@echo "  build-player - Build the musicplayerd daemon"
	@echo "  run-player   - Run musicplayerd locally"
	@echo "  cap-build    - Build frontend and sync to iOS/Android"
	@echo "  cap-sync     - Sync web assets to native platforms"
	@echo "  cap-open-ios - Open iOS project in Xcode"
	@echo "  cap-open-android - Open Android project in Android Studio"
	@echo "  cap-run-ios  - Build and run on iOS device/simulator"
	@echo "  cap-run-android - Build and run on Android device/emulator"
	@echo "  docker-build - Build Docker containers"
	@echo "  docker-up    - Start Docker containers"
	@echo "  docker-down  - Stop Docker containers"

# Install dependencies
.PHONY: deps
deps:
	go mod download

# Generate all icons and splash screens
.PHONY: icon
icon:
	go run ./cmd/icongen -o ./cmd/musicd/static/icons -name waveform
	@echo "Copying web icons..."
	@cp -f ./cmd/musicd/static/icons/waveform.ico ./cmd/musicd/static/favicon.ico
	@cp -rf ./cmd/musicd/static/icons/ ./frontend/assets/
	@cp -f ./cmd/musicd/static/icons/waveform.ico ./frontend/public/favicon.ico
	@echo "Copying PWA icons..."
	@cp -f ./cmd/musicd/static/icons/pwa/icon-192.png ./frontend/public/icon-192.png
	@cp -f ./cmd/musicd/static/icons/pwa/icon-512.png ./frontend/public/icon-512.png
	@cp -f ./cmd/musicd/static/icons/pwa/icon-192-maskable.png ./frontend/public/icon-192-maskable.png
	@cp -f ./cmd/musicd/static/icons/pwa/icon-512-maskable.png ./frontend/public/icon-512-maskable.png
	@cp -f ./cmd/musicd/static/icons/pwa/apple-touch-icon.png ./frontend/public/apple-touch-icon.png
	@echo "Copying iOS Capacitor icon..."
	@cp -f ./cmd/musicd/static/icons/ios/AppIcon-512@2x.png ./frontend/ios/App/App/Assets.xcassets/AppIcon.appiconset/AppIcon-512@2x.png
	@echo "Copying Android Capacitor icons..."
	@for density in mdpi hdpi xhdpi xxhdpi xxxhdpi; do \
		cp -f ./cmd/musicd/static/icons/android/mipmap-$$density/ic_launcher.png ./frontend/android/app/src/main/res/mipmap-$$density/ic_launcher.png; \
		cp -f ./cmd/musicd/static/icons/android/mipmap-$$density/ic_launcher_round.png ./frontend/android/app/src/main/res/mipmap-$$density/ic_launcher_round.png; \
		cp -f ./cmd/musicd/static/icons/android/mipmap-$$density/ic_launcher_foreground.png ./frontend/android/app/src/main/res/mipmap-$$density/ic_launcher_foreground.png; \
	done
	@echo "Copying iOS splash screens..."
	@cp -f ./cmd/musicd/static/icons/splash/ios/splash-2732x2732.png ./frontend/ios/App/App/Assets.xcassets/Splash.imageset/splash-2732x2732.png
	@cp -f ./cmd/musicd/static/icons/splash/ios/splash-2732x2732-1.png ./frontend/ios/App/App/Assets.xcassets/Splash.imageset/splash-2732x2732-1.png
	@cp -f ./cmd/musicd/static/icons/splash/ios/splash-2732x2732-2.png ./frontend/ios/App/App/Assets.xcassets/Splash.imageset/splash-2732x2732-2.png
	@echo "Copying Android splash screens..."
	@for variant in drawable drawable-port-mdpi drawable-port-hdpi drawable-port-xhdpi drawable-port-xxhdpi drawable-port-xxxhdpi drawable-land-mdpi drawable-land-hdpi drawable-land-xhdpi drawable-land-xxhdpi drawable-land-xxxhdpi; do \
		cp -f ./cmd/musicd/static/icons/splash/android/$$variant/splash.png ./frontend/android/app/src/main/res/$$variant/splash.png; \
	done
	@echo "All icons and splash screens generated and copied!"

# Build the frontend
.PHONY: frontend
frontend:
	@cd ./frontend && corepack yarn install && corepack yarn generate
	-@rm -rf ./cmd/musicd/ui/
	@mkdir -p ./cmd/musicd/ui/
	@cp -rf ./frontend/.output/public/* ./cmd/musicd/ui/

# Build the application
.PHONY: build
build:
	@mkdir -p ./bin
	go build -o ./bin/musicd ./cmd/musicd

# Build the musicplayerd daemon
.PHONY: build-player
build-player:
	@mkdir -p ./bin
	go build -o ./bin/musicplayerd ./cmd/musicplayerd

# Run musicplayerd locally
.PHONY: run-player
run-player:
	go run ./cmd/musicplayerd

# Run the application locally
.PHONY: run
run:
	go run ./cmd/musicd

# Development mode with file watching (requires air)
.PHONY: dev
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Installing air for hot reload..."; \
		go install github.com/air-verse/air@latest; \
		air; \
	fi

.PHONY: frontend-dev
frontend-dev:
	@cd ./frontend && corepack yarn install && corepack yarn dev

# Run tests
.PHONY: test
test:
	go test -v ./...

# Run frontend e2e tests (requires dev server or backend running)
.PHONY: test-e2e
test-e2e:
	@cd ./frontend && corepack yarn install && corepack yarn test:e2e

# Run frontend e2e tests with browser visible
.PHONY: test-e2e-headed
test-e2e-headed:
	@cd ./frontend && corepack yarn install && corepack yarn test:e2e:headed

# Open Playwright HTML report from last run
.PHONY: test-e2e-report
test-e2e-report:
	@cd ./frontend && corepack yarn test:e2e:report

# Clean build artifacts
.PHONY: clean
clean: db-down docker-down
	rm -rf bin/

# Docker commands
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down --remove-orphans

docker-logs:
	docker-compose logs -f

# Database commands
db-up:
	docker-compose up -d db

db-down:
	docker-compose stop db

db-reset:
	docker-compose down --remove-orphans -v
	docker-compose up -d db

# Capacitor - build frontend and sync to native platforms
.PHONY: cap-build
cap-build: frontend
	@cd ./frontend && npx cap sync

# Capacitor - sync web assets only (after manual frontend build)
.PHONY: cap-sync
cap-sync:
	@cd ./frontend && npx cap sync

# Capacitor - open iOS project in Xcode
.PHONY: cap-open-ios
cap-open-ios:
	@cd ./frontend && npx cap open ios

# Capacitor - open Android project in Android Studio
.PHONY: cap-open-android
cap-open-android:
	@cd ./frontend && npx cap open android

# Capacitor - build and run on iOS
.PHONY: cap-run-ios
cap-run-ios: cap-build
	@cd ./frontend && npx cap run ios

# Capacitor - build and run on Android
.PHONY: cap-run-android
cap-run-android: cap-build
	@cd ./frontend && npx cap run android

# Electron - development mode
.PHONY: electron-dev
electron-dev: frontend
	@cd ./frontend && NODE_ENV=development corepack yarn electron:dev

# Electron - build desktop application
.PHONY: electron-build
electron-build:
	@cd ./frontend && corepack yarn electron:build

# Format code
fmt:
	go fmt ./...
	cd ./frontend && corepack yarn install && corepack yarn prettier
