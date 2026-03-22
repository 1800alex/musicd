package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	cfg := LoadConfig()

	url := flag.String("url", cfg.ServerURL, "musicd server URL")
	session := flag.String("session", "", "session ID to auto-connect (skip session picker)")
	pageSize := flag.Int("page-size", 50, "items per page in browse views")
	flag.Parse()

	app := NewApp(*url, *pageSize)

	// Register all pages
	app.RegisterPage("connect", NewConnectPage(app))
	app.RegisterPage("tracks", NewTracksPage(app))
	app.RegisterPage("artists", NewArtistsPage(app))
	app.RegisterPage("playlists", NewPlaylistsPage(app))
	app.RegisterPage("nowplaying", NewNowPlayingPage(app))

	if *session != "" {
		app.ConnectToSession(*session, "")
		app.NavigateTo("tracks")
	} else {
		app.NavigateTo("connect")
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
