package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// StatusBar displays now-playing info at the bottom of the screen.
type StatusBar struct {
	*tview.Flex
	trackInfo *tview.TextView
	progress  *tview.TextView
	controls  *tview.TextView
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	trackInfo := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	trackInfo.SetBackgroundColor(tcell.ColorDarkSlateGray)

	progress := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	progress.SetBackgroundColor(tcell.ColorDarkSlateGray)

	controls := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	controls.SetBackgroundColor(tcell.ColorDarkSlateGray)

	flex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(trackInfo, 0, 2, false).
		AddItem(progress, 0, 2, false).
		AddItem(controls, 0, 1, false)
	flex.SetBackgroundColor(tcell.ColorDarkSlateGray)

	return &StatusBar{
		Flex:      flex,
		trackInfo: trackInfo,
		progress:  progress,
		controls:  controls,
	}
}

// Update refreshes the status bar from a player state.
func (s *StatusBar) Update(state *PlayerState) {
	if state == nil {
		s.trackInfo.SetText(" [gray]No session connected")
		s.progress.SetText("")
		s.controls.SetText("")
		return
	}

	// Track info
	if state.CurrentTrack != nil {
		t := state.CurrentTrack
		title := t.Title
		if title == "" {
			title = t.Filename
		}
		s.trackInfo.SetText(fmt.Sprintf(" [white]%s[-] - [yellow]%s[-]", t.Artist, title))
	} else {
		s.trackInfo.SetText(" [gray]No track playing")
	}

	// Progress bar
	currentStr := formatDuration(state.CurrentTime)
	durationStr := formatDuration(state.Duration)

	barWidth := 20
	filled := 0
	if state.Duration > 0 {
		filled = int((state.CurrentTime / state.Duration) * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
	}
	bar := "[green]" + strings.Repeat("=", filled) + ">" + "[-]" + strings.Repeat("-", barWidth-filled)
	s.progress.SetText(fmt.Sprintf("%s [%s] %s", currentStr, bar, durationStr))

	// Controls
	playIcon := "[green]>[-]"
	if !state.IsPlaying {
		playIcon = "[yellow]||[-]"
	}

	shuffleStr := ""
	if state.Shuffle {
		shuffleStr = " [cyan]S[-]"
	}

	repeatStr := ""
	switch state.RepeatMode {
	case "One":
		repeatStr = " [magenta]R:1[-]"
	case "All":
		repeatStr = " [magenta]R:A[-]"
	}

	volStr := fmt.Sprintf("Vol:%d", int(state.Volume))
	if state.Muted {
		volStr = "[red]MUTED[-]"
	}

	s.controls.SetText(fmt.Sprintf("%s%s%s  %s ", playIcon, shuffleStr, repeatStr, volStr))
}

func formatDuration(seconds float64) string {
	total := int(seconds)
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
