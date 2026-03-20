package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NowPlayingPage shows the current track with cover art.
type NowPlayingPage struct {
	*tview.Flex
	app        *App
	coverImage *tview.Image
	trackInfo  *tview.TextView
	mu         sync.Mutex
	lastArtID  string // cached cover art ID to avoid re-fetching
}

// NewNowPlayingPage creates a new Now Playing page.
func NewNowPlayingPage(app *App) *NowPlayingPage {
	p := &NowPlayingPage{
		app: app,
	}

	p.coverImage = tview.NewImage()
	p.coverImage.SetBackgroundColor(tcell.ColorDefault)

	p.trackInfo = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	p.trackInfo.SetBorder(true).SetTitle(" Now Playing ")
	p.trackInfo.SetBackgroundColor(tcell.ColorDefault)

	p.Flex = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(p.coverImage, 0, 1, false).
		AddItem(p.trackInfo, 0, 1, false)

	p.setupKeys()
	return p
}

func (p *NowPlayingPage) setupKeys() {
	p.trackInfo.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			p.app.GoBack()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'h' {
				p.app.GoBack()
				return nil
			}
		}
		return event
	})
}

// Load refreshes the now-playing display from the current state.
func (p *NowPlayingPage) Load() {
	state := p.app.GetState()
	if state == nil || state.CurrentTrack == nil {
		p.trackInfo.SetText("\n  [gray]No track playing[-]")
		p.coverImage.SetImage(nil)
		p.mu.Lock()
		p.lastArtID = ""
		p.mu.Unlock()
		return
	}

	t := state.CurrentTrack
	title := t.Title
	if title == "" {
		title = t.Filename
	}

	playState := "[green]Playing[-]"
	if !state.IsPlaying {
		playState = "[yellow]Paused[-]"
	}

	info := fmt.Sprintf("\n  [white::b]%s[-:-:-]\n\n  [yellow]%s[-]\n  [gray]%s[-]",
		title, t.Artist, t.Album)
	if t.Year > 0 {
		info += fmt.Sprintf(" [gray](%d)[-]", t.Year)
	}
	// Use interpolated progress for smooth time display
	currentTime, duration := p.app.GetInterpolatedProgress()

	info += fmt.Sprintf("\n\n  %s  %s / %s",
		playState,
		formatDuration(currentTime),
		formatDuration(duration))

	volStr := fmt.Sprintf("Vol: %d%%", int(state.Volume))
	if state.Muted {
		volStr = "[red]MUTED[-]"
	}
	info += fmt.Sprintf("\n  %s", volStr)

	if state.Shuffle {
		info += "  [cyan]Shuffle[-]"
	}
	if state.RepeatMode != "" && state.RepeatMode != "Off" {
		info += fmt.Sprintf("  [magenta]Repeat: %s[-]", state.RepeatMode)
	}

	p.trackInfo.SetText(info)

	// Fetch cover art if track has one and it changed
	artID := t.CoverArtID
	p.mu.Lock()
	changed := artID != p.lastArtID
	p.lastArtID = artID
	p.mu.Unlock()

	if changed {
		if artID == "" {
			p.coverImage.SetImage(nil)
		} else {
			go p.fetchCoverArt(artID)
		}
	}
}

func (p *NowPlayingPage) fetchCoverArt(id string) {
	data, err := p.app.client.GetCoverArt(id)
	if err != nil || len(data) == 0 {
		p.app.tviewApp.QueueUpdateDraw(func() {
			p.coverImage.SetImage(nil)
		})
		return
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		p.app.tviewApp.QueueUpdateDraw(func() {
			p.coverImage.SetImage(nil)
		})
		return
	}

	p.app.tviewApp.QueueUpdateDraw(func() {
		// Verify the art ID hasn't changed while we were fetching
		p.mu.Lock()
		current := p.lastArtID
		p.mu.Unlock()
		if current == id {
			p.coverImage.SetImage(img)
		}
	})
}
