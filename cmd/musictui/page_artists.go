package main

import (
	"fmt"
	"musicd/lib/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ArtistsPage displays a paginated, searchable list of artists.
type ArtistsPage struct {
	*tview.Flex
	table      *tview.Table
	searchBar  *tview.InputField
	statusLine *tview.TextView
	pagination *PaginationState
	app        *App
	artists    []types.Artist
	searching  bool
}

// NewArtistsPage creates a new artists browse page.
func NewArtistsPage(app *App) *ArtistsPage {
	p := &ArtistsPage{
		app:        app,
		pagination: NewPaginationState(app.pageSize),
	}

	p.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.table.SetBorder(true).SetTitle(" Artists ")

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

func (p *ArtistsPage) showSearch() {
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

func (p *ArtistsPage) hideSearch() {
	p.searching = false
	p.Flex.Clear()
	p.Flex.AddItem(p.table, 0, 1, true).
		AddItem(p.statusLine, 1, 0, false)
	p.app.tviewApp.SetFocus(p.table)
}

func (p *ArtistsPage) setupKeys() {
	p.table.SetSelectedFunc(func(row, col int) {
		p.selectArtist(row)
	})

	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if p.pagination.Search != "" {
				p.pagination.ClearSearch()
				p.Load()
			} else {
				p.app.GoBack()
			}
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
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
				p.selectArtist(row)
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

func (p *ArtistsPage) selectArtist(row int) {
	idx := row - 1
	if idx < 0 || idx >= len(p.artists) {
		return
	}
	artist := p.artists[idx]

	// Clean up old detail page, create new one
	p.app.pages.RemovePage("artist-detail")
	detail := NewArtistDetailPage(p.app, artist.ID, artist.Name)
	p.app.pages.AddPage("artist-detail", detail, true, false)
	p.app.pageMap["artist-detail"] = detail
	p.app.NavigateTo("artist-detail")
}

// Load fetches artists from the server.
func (p *ArtistsPage) Load() {
	p.statusLine.SetText("[yellow]Loading...")
	go func() {
		resp, artists, err := p.app.client.GetArtists(
			p.pagination.Page, p.pagination.PageSize, p.pagination.Search,
		)
		p.app.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				p.statusLine.SetText(fmt.Sprintf("[red]Error: %v", err))
				return
			}
			p.pagination.UpdateFromResponse(resp.Page, resp.TotalPages, resp.Total)
			p.artists = artists
			p.renderTable()
			p.statusLine.SetText(fmt.Sprintf("[white]%s  |  /: search  Enter/l: open  [] ←→: pages  h: back",
				p.pagination.StatusText()))
		})
	}()
}

func (p *ArtistsPage) renderTable() {
	p.table.Clear()

	headers := []string{"#", "Name", "Tracks"}
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

	offset := (p.pagination.Page - 1) * p.pagination.PageSize
	for i, a := range p.artists {
		row := i + 1
		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", offset+i+1)).
			SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 1, tview.NewTableCell(a.Name).SetExpansion(1))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", a.TrackCount)).
			SetAlign(tview.AlignRight))
	}

	if len(p.artists) > 0 {
		p.table.Select(1, 0)
	}
}
