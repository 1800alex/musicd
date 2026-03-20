package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	url := flag.String("url", "http://localhost:8080", "musicd server URL")
	session := flag.String("session", "", "session ID to auto-connect (skip session picker)")
	pageSize := flag.Int("page-size", 50, "items per page in browse views")
	flag.Parse()

	app := NewApp(*url, *pageSize)

	// Register all pages
	app.RegisterPage("sessions", NewSessionsPage(app))
	app.RegisterPage("connect", NewConnectPage(app))
	app.RegisterPage("tracks", NewTracksPage(app))
	app.RegisterPage("artists", NewArtistsPage(app))
	app.RegisterPage("playlists", NewPlaylistsPage(app))
	app.RegisterPage("queue", NewQueuePage(app))
	app.RegisterPage("nowplaying", NewNowPlayingPage(app))

	if *session != "" {
		app.ConnectToSession(*session, "")
		app.NavigateTo("tracks")
	} else {
		app.NavigateTo("sessions")
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
