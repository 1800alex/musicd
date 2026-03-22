package main

import (
	"fmt"
	"musicd/lib/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// AlbumDetailPage displays an album's tracks.
type AlbumDetailPage struct {
	*tview.Flex
	table   *tview.Table
	status  *tview.TextView
	app     *App
	albumID string
	album   *types.Album
}

// NewAlbumDetailPage creates a new album detail page.
func NewAlbumDetailPage(app *App, albumID, name string) *AlbumDetailPage {
	p := &AlbumDetailPage{
		app:     app,
		albumID: albumID,
	}

	title := " Album "
	if name != "" {
		title = fmt.Sprintf(" %s ", name)
	}

	p.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.table.SetBorder(true).SetTitle(title)

	p.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.table, 0, 1, true).
		AddItem(p.status, 1, 0, false)

	p.setupKeys()
	return p
}

func (p *AlbumDetailPage) setupKeys() {
	p.table.SetSelectedFunc(func(row, col int) {
		p.selectTrack(row)
	})

	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyBackspace, tcell.KeyBackspace2:
			p.app.GoBack()
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
			case 'a':
				p.app.SendCommand("play_album", map[string]interface{}{
					"id": p.albumID,
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

func (p *AlbumDetailPage) selectTrack(row int) {
	if p.album == nil {
		return
	}
	idx := row - 1
	if idx < 0 || idx >= len(p.album.Tracks) {
		return
	}
	t := p.album.Tracks[idx]
	p.app.SendCommand("play_album_track", map[string]interface{}{
		"id":       t.ID,
		"album_id": p.albumID,
	})
}

// Load fetches album detail from the server.
func (p *AlbumDetailPage) Load() {
	p.status.SetText("[yellow]Loading...")
	go func() {
		album, err := p.app.client.GetAlbum(p.albumID)
		p.app.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				p.status.SetText(fmt.Sprintf("[red]Error: %v", err))
				return
			}
			p.album = album
			title := album.Name
			if album.Artist != "" {
				title = fmt.Sprintf("%s - %s", album.Artist, album.Name)
			}
			if album.Year > 0 {
				title = fmt.Sprintf("%s (%d)", title, album.Year)
			}
			p.table.SetTitle(fmt.Sprintf(" %s ", title))
			p.renderTable()
			p.status.SetText(fmt.Sprintf("[white]%d tracks  |  Enter/l: play  a: play all  h: back",
				len(album.Tracks)))
		})
	}()
}

func (p *AlbumDetailPage) renderTable() {
	p.table.Clear()

	headers := []string{"#", "Title", "Duration"}
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

	if p.album == nil {
		return
	}

	for i, t := range p.album.Tracks {
		row := i + 1
		title := t.Title
		if title == "" {
			title = t.Filename
		}
		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 1, tview.NewTableCell(title).SetExpansion(1))
		p.table.SetCell(row, 2, tview.NewTableCell(t.Duration).SetAlign(tview.AlignRight))
	}

	if len(p.album.Tracks) > 0 {
		p.table.Select(1, 0)
	}
}
