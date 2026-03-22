package main

import (
	"fmt"
	"musicd/lib/types"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PlaylistsPage displays the list of playlists.
type PlaylistsPage struct {
	*tview.Flex
	table     *tview.Table
	searchBar *tview.InputField
	status    *tview.TextView
	app       *App
	playlists []types.Playlist
	filtered  []types.Playlist
	search    string
	searching bool
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

	p.searchBar = tview.NewInputField().
		SetLabel(" Search: ").
		SetFieldWidth(40)
	p.searchBar.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			p.search = p.searchBar.GetText()
			p.hideSearch()
			p.applyFilter()
		case tcell.KeyEscape:
			if p.search != "" {
				p.search = ""
				p.applyFilter()
			}
			p.hideSearch()
		}
	})

	p.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.table, 0, 1, true).
		AddItem(p.status, 1, 0, false)

	p.setupKeys()
	return p
}

func (p *PlaylistsPage) showSearch() {
	if p.searching {
		return
	}
	p.searching = true
	p.searchBar.SetText(p.search)
	p.Flex.Clear()
	p.Flex.AddItem(p.searchBar, 1, 0, true).
		AddItem(p.table, 0, 1, false).
		AddItem(p.status, 1, 0, false)
	p.app.tviewApp.SetFocus(p.searchBar)
}

func (p *PlaylistsPage) hideSearch() {
	p.searching = false
	p.Flex.Clear()
	p.Flex.AddItem(p.table, 0, 1, true).
		AddItem(p.status, 1, 0, false)
	p.app.tviewApp.SetFocus(p.table)
}

func (p *PlaylistsPage) applyFilter() {
	if p.search == "" {
		p.filtered = p.playlists
	} else {
		q := strings.ToLower(p.search)
		p.filtered = nil
		for _, pl := range p.playlists {
			if strings.Contains(strings.ToLower(pl.Name), q) ||
				strings.Contains(strings.ToLower(pl.Description), q) {
				p.filtered = append(p.filtered, pl)
			}
		}
	}
	p.renderTable()
	p.updateStatus()
}

func (p *PlaylistsPage) updateStatus() {
	searchHint := ""
	if p.search != "" {
		searchHint = fmt.Sprintf("  filter: \"%s\"", p.search)
	}
	p.status.SetText(fmt.Sprintf("[white]%d playlists%s  |  /: search  Enter/l: open  h: back",
		len(p.filtered), searchHint))
}

func (p *PlaylistsPage) setupKeys() {
	p.table.SetSelectedFunc(func(row, col int) {
		p.selectPlaylist(row)
	})

	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if p.search != "" {
				p.search = ""
				p.applyFilter()
			} else {
				p.app.GoBack()
			}
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
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
			case '/':
				p.showSearch()
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
	if idx < 0 || idx >= len(p.filtered) {
		return
	}
	pl := p.filtered[idx]

	// Clean up old detail page, create new one
	p.app.pages.RemovePage("playlist-detail")
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
			p.applyFilter()
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

	for i, pl := range p.filtered {
		row := i + 1
		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 1, tview.NewTableCell(pl.Name).SetExpansion(1))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", pl.TrackCount)).
			SetAlign(tview.AlignRight))
		p.table.SetCell(row, 3, tview.NewTableCell(pl.Description).
			SetExpansion(1).SetTextColor(tcell.ColorGray))
	}

	if len(p.filtered) > 0 {
		p.table.Select(1, 0)
	}
}
