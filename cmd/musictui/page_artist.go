package main

import (
	"fmt"
	"musicd/lib/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ArtistDetailPage displays an artist's albums and tracks.
type ArtistDetailPage struct {
	*tview.Flex
	table    *tview.Table
	status   *tview.TextView
	app      *App
	artistID string
	name     string
	artist   *types.Artist
	// Row mapping: row index -> what it represents
	rowMap []artistRow
}

type artistRow struct {
	isAlbum bool
	albumID string
	track   *types.Track
}

// NewArtistDetailPage creates a new artist detail page.
func NewArtistDetailPage(app *App, artistID, name string) *ArtistDetailPage {
	p := &ArtistDetailPage{
		app:      app,
		artistID: artistID,
		name:     name,
	}

	p.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.table.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", name))

	p.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.table, 0, 1, true).
		AddItem(p.status, 1, 0, false)

	p.setupKeys()
	return p
}

func (p *ArtistDetailPage) setupKeys() {
	p.table.SetSelectedFunc(func(row, col int) {
		p.selectRow(row)
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
				p.selectRow(row)
				return nil
			case 'h':
				p.app.GoBack()
				return nil
			case 'a':
				p.app.SendCommand("play_artist", map[string]interface{}{
					"id": p.artistID,
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

func (p *ArtistDetailPage) selectRow(row int) {
	idx := row - 1 // skip header
	if idx < 0 || idx >= len(p.rowMap) {
		return
	}
	r := p.rowMap[idx]
	if r.isAlbum {
		// Open album detail
		detail := NewAlbumDetailPage(p.app, r.albumID, "")
		p.app.pages.AddPage("album-detail", detail, true, false)
		p.app.pageMap["album-detail"] = detail
		p.app.NavigateTo("album-detail")
	} else if r.track != nil {
		p.app.SendCommand("play_artist_track", map[string]interface{}{
			"id":        r.track.ID,
			"artist_id": p.artistID,
		})
	}
}

// Load fetches artist detail from the server.
func (p *ArtistDetailPage) Load() {
	p.status.SetText("[yellow]Loading...")
	go func() {
		artist, err := p.app.client.GetArtist(p.artistID)
		p.app.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				p.status.SetText(fmt.Sprintf("[red]Error: %v", err))
				return
			}
			p.artist = artist
			p.name = artist.Name
			p.table.SetTitle(fmt.Sprintf(" %s ", artist.Name))
			p.renderTable()
			p.status.SetText(fmt.Sprintf("[white]%d tracks  |  Enter/l: play/open  a: play all  h: back",
				artist.TrackCount))
		})
	}()
}

func (p *ArtistDetailPage) renderTable() {
	p.table.Clear()
	p.rowMap = nil

	headers := []string{"", "Title", "Album", "Duration"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		if i == 0 || i == 3 {
			cell.SetExpansion(0)
		}
		p.table.SetCell(0, i, cell)
	}

	row := 1
	if p.artist == nil {
		return
	}

	for _, album := range p.artist.Albums {
		// Album header row
		albumName := album.Name
		if album.Year > 0 {
			albumName = fmt.Sprintf("%s (%d)", album.Name, album.Year)
		}
		p.table.SetCell(row, 0, tview.NewTableCell("").SetSelectable(true))
		p.table.SetCell(row, 1, tview.NewTableCell(albumName).
			SetTextColor(tcell.ColorGreen).SetExpansion(1).SetSelectable(true))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d tracks", album.TrackCount)).
			SetTextColor(tcell.ColorGreen).SetExpansion(1).SetSelectable(true))
		p.table.SetCell(row, 3, tview.NewTableCell("").SetSelectable(true))
		p.rowMap = append(p.rowMap, artistRow{isAlbum: true, albumID: album.ID})
		row++

		// Tracks under this album
		for i := range album.Tracks {
			t := &album.Tracks[i]
			title := t.Title
			if title == "" {
				title = t.Filename
			}
			p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("  %d", i+1)).
				SetTextColor(tcell.ColorGray))
			p.table.SetCell(row, 1, tview.NewTableCell("  "+title).SetExpansion(1))
			p.table.SetCell(row, 2, tview.NewTableCell("").SetExpansion(1))
			p.table.SetCell(row, 3, tview.NewTableCell(t.Duration).SetAlign(tview.AlignRight))
			p.rowMap = append(p.rowMap, artistRow{track: t})
			row++
		}
	}

	if row > 1 {
		p.table.Select(1, 0)
	}
}
