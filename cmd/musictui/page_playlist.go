package main

import (
	"fmt"
	"musicd/lib/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PlaylistDetailPage displays tracks in a playlist with pagination and search.
type PlaylistDetailPage struct {
	*tview.Flex
	table      *tview.Table
	searchBar  *tview.InputField
	statusLine *tview.TextView
	pagination *PaginationState
	app        *App
	playlistID string
	name       string
	tracks     []types.Track
	searching  bool
}

// NewPlaylistDetailPage creates a new playlist detail page.
func NewPlaylistDetailPage(app *App, playlistID, name string) *PlaylistDetailPage {
	p := &PlaylistDetailPage{
		app:        app,
		playlistID: playlistID,
		name:       name,
		pagination: NewPaginationState(app.pageSize),
	}

	p.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.table.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", name))

	p.searchBar = tview.NewInputField().
		SetLabel(" Search: ").
		SetFieldWidth(40)
	p.searchBar.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			p.pagination.SetSearch(p.searchBar.GetText())
			p.hideSearch()
			p.Load()
		case tcell.KeyEscape:
			if p.pagination.Search != "" {
				p.pagination.ClearSearch()
				p.Load()
			}
			p.hideSearch()
		}
	})

	p.statusLine = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.table, 0, 1, true).
		AddItem(p.statusLine, 1, 0, false)

	p.setupKeys()
	return p
}

func (p *PlaylistDetailPage) showSearch() {
	if p.searching {
		return
	}
	p.searching = true
	p.searchBar.SetText(p.pagination.Search)
	p.Flex.Clear()
	p.Flex.AddItem(p.searchBar, 1, 0, true).
		AddItem(p.table, 0, 1, false).
		AddItem(p.statusLine, 1, 0, false)
	p.app.tviewApp.SetFocus(p.searchBar)
}

func (p *PlaylistDetailPage) hideSearch() {
	p.searching = false
	p.Flex.Clear()
	p.Flex.AddItem(p.table, 0, 1, true).
		AddItem(p.statusLine, 1, 0, false)
	p.app.tviewApp.SetFocus(p.table)
}

func (p *PlaylistDetailPage) setupKeys() {
	p.table.SetSelectedFunc(func(row, col int) {
		p.selectTrack(row)
	})

	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			p.app.GoBack()
			return nil
		case tcell.KeyLeft:
			if p.pagination.PrevPage() {
				p.Load()
			}
			return nil
		case tcell.KeyRight:
			if p.pagination.NextPage() {
				p.Load()
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'l':
				row, _ := p.table.GetSelection()
				p.selectTrack(row)
				return nil
			case 'h':
				p.app.GoBack()
				return nil
			case '/':
				p.showSearch()
				return nil
			case '[':
				if p.pagination.PrevPage() {
					p.Load()
				}
				return nil
			case ']':
				if p.pagination.NextPage() {
					p.Load()
				}
				return nil
			case 'a':
				p.app.SendCommand("play_playlist", map[string]interface{}{
					"id":   p.playlistID,
					"name": p.name,
				})
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

func (p *PlaylistDetailPage) selectTrack(row int) {
	idx := row - 1
	if idx < 0 || idx >= len(p.tracks) {
		return
	}
	t := p.tracks[idx]
	p.app.SendCommand("play_playlist_track", map[string]interface{}{
		"id":                   t.ID,
		"playlist_id":          p.playlistID,
		"playlist_position_id": t.PlaylistPositionID,
		"playlist_name":        p.name,
	})
}

// Load fetches playlist tracks from the server.
func (p *PlaylistDetailPage) Load() {
	p.statusLine.SetText("[yellow]Loading...")
	go func() {
		resp, tracks, err := p.app.client.GetPlaylistTracks(
			p.playlistID, p.pagination.Page, p.pagination.PageSize, p.pagination.Search,
		)
		p.app.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				p.statusLine.SetText(fmt.Sprintf("[red]Error: %v", err))
				return
			}
			p.pagination.UpdateFromResponse(resp.Page, resp.TotalPages, resp.Total)
			p.tracks = tracks
			p.renderTable()
			p.statusLine.SetText(fmt.Sprintf("[white]%s  |  /: search  Enter/l: play  a: play all  [[]/[]]/<-/->: pages  h: back",
				p.pagination.StatusText()))
		})
	}()
}

func (p *PlaylistDetailPage) renderTable() {
	p.table.Clear()

	headers := []string{"#", "Title", "Artist", "Album", "Duration"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		if i == 0 || i == 4 {
			cell.SetExpansion(0)
		}
		p.table.SetCell(0, i, cell)
	}

	offset := (p.pagination.Page - 1) * p.pagination.PageSize
	for i, t := range p.tracks {
		row := i + 1
		title := t.Title
		if title == "" {
			title = t.Filename
		}
		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", offset+i+1)).
			SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 1, tview.NewTableCell(title).SetExpansion(1))
		p.table.SetCell(row, 2, tview.NewTableCell(t.Artist).SetExpansion(1))
		p.table.SetCell(row, 3, tview.NewTableCell(t.Album).SetExpansion(1))
		p.table.SetCell(row, 4, tview.NewTableCell(t.Duration).SetAlign(tview.AlignRight))
	}

	if len(p.tracks) > 0 {
		p.table.Select(1, 0)
	}
}
