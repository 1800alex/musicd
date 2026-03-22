package main

import (
	"fmt"
	"musicd/lib/types"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ArtistDetailPage displays an artist's albums (top) and tracks (bottom).
type ArtistDetailPage struct {
	*tview.Flex
	albumTable *tview.Table
	trackTable *tview.Table
	searchBar  *tview.InputField
	status     *tview.TextView
	app        *App
	artistID   string
	name       string
	artist     *types.Artist

	// Album list: index 0 = "All Tracks", then one per album
	albumEntries []albumEntry

	// Tracks currently shown in the bottom panel
	allTracks      []types.Track // unfiltered tracks for current album selection
	filteredTracks []types.Track
	search         string
	searching      bool
	selectedAlbum  int // index into albumEntries
	focusedPanel   int // 0 = albums, 1 = tracks
}

type albumEntry struct {
	id         string // empty for "All Tracks"
	name       string
	trackCount int
	tracks     []types.Track
}

// NewArtistDetailPage creates a new artist detail page.
func NewArtistDetailPage(app *App, artistID, name string) *ArtistDetailPage {
	p := &ArtistDetailPage{
		app:      app,
		artistID: artistID,
		name:     name,
	}

	p.albumTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.albumTable.SetBorder(true).SetTitle(" Albums ")

	p.trackTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.trackTable.SetBorder(true).SetTitle(" Tracks ")

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
		AddItem(p.albumTable, 10, 0, true).
		AddItem(p.trackTable, 0, 1, false).
		AddItem(p.status, 1, 0, false)

	p.setupKeys()
	return p
}

func (p *ArtistDetailPage) showSearch() {
	if p.searching {
		return
	}
	p.searching = true
	p.searchBar.SetText(p.search)
	p.Flex.Clear()
	p.Flex.
		AddItem(p.albumTable, 10, 0, false).
		AddItem(p.searchBar, 1, 0, true).
		AddItem(p.trackTable, 0, 1, false).
		AddItem(p.status, 1, 0, false)
	p.app.tviewApp.SetFocus(p.searchBar)
}

func (p *ArtistDetailPage) hideSearch() {
	p.searching = false
	p.Flex.Clear()
	p.Flex.
		AddItem(p.albumTable, 10, 0, p.focusedPanel == 0).
		AddItem(p.trackTable, 0, 1, p.focusedPanel == 1).
		AddItem(p.status, 1, 0, false)
	p.app.tviewApp.SetFocus(p.trackTable)
	p.focusedPanel = 1
}

func (p *ArtistDetailPage) focusAlbums() {
	p.focusedPanel = 0
	p.albumTable.SetBorderColor(tcell.ColorWhite)
	p.trackTable.SetBorderColor(tcell.ColorDefault)
	p.app.tviewApp.SetFocus(p.albumTable)
}

func (p *ArtistDetailPage) focusTracks() {
	if len(p.filteredTracks) == 0 {
		return
	}
	p.focusedPanel = 1
	p.albumTable.SetBorderColor(tcell.ColorDefault)
	p.trackTable.SetBorderColor(tcell.ColorWhite)
	p.app.tviewApp.SetFocus(p.trackTable)
}

func (p *ArtistDetailPage) setupKeys() {
	// Album table keys
	p.albumTable.SetSelectedFunc(func(row, col int) {
		p.selectAlbum(row)
	})

	p.albumTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyBackspace, tcell.KeyBackspace2:
			p.app.GoBack()
			return nil
		case tcell.KeyTab:
			if len(p.filteredTracks) > 0 {
				p.focusTracks()
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'l':
				row, _ := p.albumTable.GetSelection()
				p.selectAlbum(row)
				return nil
			case 'h':
				p.app.GoBack()
				return nil
			case 'a':
				p.app.SendCommand("play_artist", map[string]interface{}{
					"id": p.artistID,
				})
				return nil
			case '/':
				p.showSearch()
				return nil
			case 'G':
				if count := p.albumTable.GetRowCount(); count > 1 {
					p.albumTable.Select(count-1, 0)
				}
				return nil
			case 'g':
				p.albumTable.Select(1, 0)
				return nil
			}
		}
		return event
	})

	// Track table keys
	p.trackTable.SetSelectedFunc(func(row, col int) {
		p.selectTrack(row)
	})

	p.trackTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			p.app.GoBack()
			return nil
		case tcell.KeyEscape:
			if p.search != "" {
				p.search = ""
				p.applyFilter()
			} else {
				p.focusAlbums()
			}
			return nil
		case tcell.KeyTab:
			p.focusAlbums()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'l':
				row, _ := p.trackTable.GetSelection()
				p.selectTrack(row)
				return nil
			case 'h':
				p.focusAlbums()
				return nil
			case 'a':
				p.app.SendCommand("play_artist", map[string]interface{}{
					"id": p.artistID,
				})
				return nil
			case '/':
				p.showSearch()
				return nil
			case 'G':
				if count := p.trackTable.GetRowCount(); count > 1 {
					p.trackTable.Select(count-1, 0)
				}
				return nil
			case 'g':
				p.trackTable.Select(1, 0)
				return nil
			}
		}
		return event
	})
}

func (p *ArtistDetailPage) selectAlbum(row int) {
	idx := row - 1 // skip header
	if idx < 0 || idx >= len(p.albumEntries) {
		return
	}
	p.selectedAlbum = idx
	p.search = ""
	p.allTracks = p.albumEntries[idx].tracks
	p.applyFilter()
	p.updateAlbumHighlight()
	if len(p.filteredTracks) > 0 {
		p.focusTracks()
	}
}

func (p *ArtistDetailPage) selectTrack(row int) {
	idx := row - 1
	if idx < 0 || idx >= len(p.filteredTracks) {
		return
	}
	t := p.filteredTracks[idx]

	entry := p.albumEntries[p.selectedAlbum]
	if entry.id == "" {
		// "All Tracks" - play as artist track
		p.app.SendCommand("play_artist_track", map[string]interface{}{
			"id":        t.ID,
			"artist_id": p.artistID,
		})
	} else {
		// Specific album
		p.app.SendCommand("play_album_track", map[string]interface{}{
			"id":       t.ID,
			"album_id": entry.id,
		})
	}
}

func (p *ArtistDetailPage) applyFilter() {
	if p.search == "" {
		p.filteredTracks = p.allTracks
	} else {
		q := strings.ToLower(p.search)
		p.filteredTracks = nil
		for _, t := range p.allTracks {
			title := t.Title
			if title == "" {
				title = t.Filename
			}
			if strings.Contains(strings.ToLower(title), q) ||
				strings.Contains(strings.ToLower(t.Album), q) {
				p.filteredTracks = append(p.filteredTracks, t)
			}
		}
	}
	p.renderTrackTable()
	p.updateStatus()
}

func (p *ArtistDetailPage) updateStatus() {
	searchHint := ""
	if p.search != "" {
		searchHint = fmt.Sprintf("  filter: \"%s\"", p.search)
	}
	albumName := "All Tracks"
	if p.selectedAlbum < len(p.albumEntries) {
		albumName = p.albumEntries[p.selectedAlbum].name
	}
	p.status.SetText(fmt.Sprintf("[white]%s: %d tracks%s  |  /: search  Tab: switch  a: play all  h: back",
		albumName, len(p.filteredTracks), searchHint))
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
			p.albumTable.SetTitle(fmt.Sprintf(" %s - Albums ", artist.Name))
			p.buildAlbumEntries()
			p.renderAlbumTable()

			// Default to "All Tracks"
			p.selectedAlbum = 0
			if len(p.albumEntries) > 0 {
				p.allTracks = p.albumEntries[0].tracks
			}
			p.search = ""
			p.applyFilter()
			p.updateAlbumHighlight()
			p.focusAlbums()
		})
	}()
}

func (p *ArtistDetailPage) buildAlbumEntries() {
	p.albumEntries = nil

	// First entry: All Tracks
	allTracks := make([]types.Track, len(p.artist.Tracks))
	copy(allTracks, p.artist.Tracks)
	p.albumEntries = append(p.albumEntries, albumEntry{
		name:       "All Tracks",
		trackCount: len(allTracks),
		tracks:     allTracks,
	})

	// One entry per album
	for _, album := range p.artist.Albums {
		tracks := make([]types.Track, len(album.Tracks))
		copy(tracks, album.Tracks)
		name := album.Name
		if album.Year > 0 {
			name = fmt.Sprintf("%s (%d)", album.Name, album.Year)
		}
		p.albumEntries = append(p.albumEntries, albumEntry{
			id:         album.ID,
			name:       name,
			trackCount: len(album.Tracks),
			tracks:     tracks,
		})
	}
}

func (p *ArtistDetailPage) renderAlbumTable() {
	p.albumTable.Clear()

	headers := []string{"", "Album", "Tracks"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		if i == 0 || i == 2 {
			cell.SetExpansion(0)
		}
		p.albumTable.SetCell(0, i, cell)
	}

	for i, entry := range p.albumEntries {
		row := i + 1
		marker := "  "
		if i == p.selectedAlbum {
			marker = "▸ "
		}
		p.albumTable.SetCell(row, 0, tview.NewTableCell(marker).
			SetTextColor(tcell.ColorGreen))
		nameColor := tcell.ColorWhite
		if i == 0 {
			nameColor = tcell.ColorAqua
		}
		p.albumTable.SetCell(row, 1, tview.NewTableCell(entry.name).
			SetExpansion(1).SetTextColor(nameColor))
		p.albumTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", entry.trackCount)).
			SetAlign(tview.AlignRight))
	}

	if len(p.albumEntries) > 0 {
		p.albumTable.ScrollToBeginning()
		p.albumTable.Select(p.selectedAlbum+1, 0)
	}
}

func (p *ArtistDetailPage) updateAlbumHighlight() {
	for i := range p.albumEntries {
		row := i + 1
		marker := "  "
		if i == p.selectedAlbum {
			marker = "▸ "
		}
		if cell := p.albumTable.GetCell(row, 0); cell != nil {
			cell.SetText(marker)
		}
	}
}

func (p *ArtistDetailPage) renderTrackTable() {
	p.trackTable.Clear()

	currentTrackID := ""
	if state := p.app.GetState(); state != nil && state.CurrentTrack != nil {
		currentTrackID = state.CurrentTrack.ID
	}

	headers := []string{"", "#", "Title", "Album", "Duration"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		if i == 0 || i == 1 || i == 4 {
			cell.SetExpansion(0)
		}
		p.trackTable.SetCell(0, i, cell)
	}

	for i, t := range p.filteredTracks {
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

		p.trackTable.SetCell(row, 0, tview.NewTableCell(indicator).SetTextColor(tcell.ColorGreen))
		p.trackTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		p.trackTable.SetCell(row, 2, tview.NewTableCell(title).SetExpansion(1).SetTextColor(textColor))
		p.trackTable.SetCell(row, 3, tview.NewTableCell(t.Album).SetExpansion(1).SetTextColor(textColor))
		p.trackTable.SetCell(row, 4, tview.NewTableCell(t.Duration).SetAlign(tview.AlignRight).SetTextColor(textColor))
	}

	if len(p.filteredTracks) > 0 {
		p.trackTable.SetSelectable(true, false)
		// Try to select the current track
		for i, t := range p.filteredTracks {
			if t.ID == currentTrackID {
				p.trackTable.Select(i+1, 0)
				return
			}
		}
		p.trackTable.Select(1, 0)
	} else {
		// Disable selection on empty table to prevent tview spin
		p.trackTable.SetSelectable(false, false)
	}
}
