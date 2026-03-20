package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// QueuePage displays the current playback queue from WebSocket state.
type QueuePage struct {
	*tview.Flex
	table  *tview.Table
	status *tview.TextView
	app    *App
	tracks []Track
}

// NewQueuePage creates a new queue view page.
func NewQueuePage(app *App) *QueuePage {
	p := &QueuePage{
		app: app,
	}

	p.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.table.SetBorder(true).SetTitle(" Queue ")

	p.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.table, 0, 1, true).
		AddItem(p.status, 1, 0, false)

	p.setupKeys()
	return p
}

func (p *QueuePage) setupKeys() {
	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			p.app.GoBack()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'h':
				p.app.GoBack()
				return nil
			case 'G':
				if count := p.table.GetRowCount(); count > 1 {
					p.table.Select(count-1, 0)
				}
				return nil
			case 'g':
				p.table.Select(1, 0)
				return nil
			}
		}
		return event
	})
}

// Load renders the queue from the current player state.
func (p *QueuePage) Load() {
	state := p.app.GetState()
	if state == nil {
		p.table.Clear()
		p.status.SetText("[gray]No session connected")
		return
	}

	p.tracks = state.TemporaryQueue
	p.renderTable(state)

	queueLen := len(state.Queue)
	totalLen := len(state.TemporaryQueue)
	p.status.SetText(fmt.Sprintf("[white]%d in priority queue, %d total  |  h: back", queueLen, totalLen))
}

func (p *QueuePage) renderTable(state *PlayerState) {
	p.table.Clear()

	headers := []string{"", "#", "Title", "Artist", "Duration"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		if i == 0 || i == 1 || i == 4 {
			cell.SetExpansion(0)
		}
		p.table.SetCell(0, i, cell)
	}

	currentTrackID := ""
	if state.CurrentTrack != nil {
		currentTrackID = state.CurrentTrack.ID
	}

	for i, t := range p.tracks {
		row := i + 1
		title := t.Title
		if title == "" {
			title = t.Filename
		}

		indicator := " "
		textColor := tcell.ColorWhite
		if t.ID == currentTrackID {
			indicator = ">"
			textColor = tcell.ColorGreen
		}

		p.table.SetCell(row, 0, tview.NewTableCell(indicator).SetTextColor(tcell.ColorGreen))
		p.table.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 2, tview.NewTableCell(title).SetExpansion(1).SetTextColor(textColor))
		p.table.SetCell(row, 3, tview.NewTableCell(t.Artist).SetExpansion(1).SetTextColor(textColor))
		p.table.SetCell(row, 4, tview.NewTableCell(t.Duration).SetAlign(tview.AlignRight).SetTextColor(textColor))
	}

	if len(p.tracks) > 0 {
		// Try to select the current track
		for i, t := range p.tracks {
			if t.ID == currentTrackID {
				p.table.Select(i+1, 0)
				return
			}
		}
		p.table.Select(1, 0)
	}
}
