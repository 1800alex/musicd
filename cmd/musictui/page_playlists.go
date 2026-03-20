package main

import (
	"fmt"
	"musicd/lib/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PlaylistsPage displays the list of playlists.
type PlaylistsPage struct {
	*tview.Flex
	table     *tview.Table
	status    *tview.TextView
	app       *App
	playlists []types.Playlist
}

// NewPlaylistsPage creates a new playlists browse page.
func NewPlaylistsPage(app *App) *PlaylistsPage {
	p := &PlaylistsPage{
		app: app,
	}

	p.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.table.SetBorder(true).SetTitle(" Playlists ")

	p.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.table, 0, 1, true).
		AddItem(p.status, 1, 0, false)

	p.setupKeys()
	return p
}

func (p *PlaylistsPage) setupKeys() {
	p.table.SetSelectedFunc(func(row, col int) {
		p.selectPlaylist(row)
	})

	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			p.app.GoBack()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'l':
				row, _ := p.table.GetSelection()
				p.selectPlaylist(row)
				return nil
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

func (p *PlaylistsPage) selectPlaylist(row int) {
	idx := row - 1
	if idx < 0 || idx >= len(p.playlists) {
		return
	}
	pl := p.playlists[idx]

	detail := NewPlaylistDetailPage(p.app, pl.ID, pl.Name)
	p.app.pages.AddPage("playlist-detail", detail, true, false)
	p.app.pageMap["playlist-detail"] = detail
	p.app.NavigateTo("playlist-detail")
}

// Load fetches playlists from the server.
func (p *PlaylistsPage) Load() {
	p.status.SetText("[yellow]Loading...")
	go func() {
		playlists, err := p.app.client.GetPlaylists()
		p.app.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				p.status.SetText(fmt.Sprintf("[red]Error: %v", err))
				return
			}
			p.playlists = playlists
			p.renderTable()
			p.status.SetText(fmt.Sprintf("[white]%d playlists  |  Enter/l: open  h: back", len(playlists)))
		})
	}()
}

func (p *PlaylistsPage) renderTable() {
	p.table.Clear()

	headers := []string{"#", "Name", "Tracks", "Description"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		if i == 0 || i == 2 {
			cell.SetExpansion(0)
		}
		p.table.SetCell(0, i, cell)
	}

	for i, pl := range p.playlists {
		row := i + 1
		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 1, tview.NewTableCell(pl.Name).SetExpansion(1))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", pl.TrackCount)).
			SetAlign(tview.AlignRight))
		p.table.SetCell(row, 3, tview.NewTableCell(pl.Description).
			SetExpansion(1).SetTextColor(tcell.ColorGray))
	}

	if len(p.playlists) > 0 {
		p.table.Select(1, 0)
	}
}
