package main

import (
	"fmt"
	"musicd/lib/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SessionsPage displays available player sessions.
type SessionsPage struct {
	*tview.Flex
	table    *tview.Table
	status   *tview.TextView
	app      *App
	sessions []types.SessionInfo
}

// NewSessionsPage creates a new session picker page.
func NewSessionsPage(app *App) *SessionsPage {
	p := &SessionsPage{
		app: app,
	}

	p.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.table.SetBorder(true).SetTitle(" Sessions ")

	p.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.table, 0, 1, true).
		AddItem(p.status, 1, 0, false)

	p.table.SetSelectedFunc(func(row, col int) {
		idx := row - 1 // skip header
		if idx >= 0 && idx < len(p.sessions) {
			s := p.sessions[idx]
			app.ConnectToSession(s.ID, s.Name)
			app.NavigateTo("tracks")
		}
	})

	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'l':
				row, _ := p.table.GetSelection()
				idx := row - 1
				if idx >= 0 && idx < len(p.sessions) {
					s := p.sessions[idx]
					app.ConnectToSession(s.ID, s.Name)
					app.NavigateTo("tracks")
				}
				return nil
			case 'R':
				p.Load()
				return nil
			}
		}
		return event
	})

	return p
}

// Load fetches sessions from the server and populates the table.
func (p *SessionsPage) Load() {
	p.status.SetText("[yellow]Loading sessions...")
	go func() {
		sessions, err := p.app.client.GetSessions()
		p.app.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				p.status.SetText(fmt.Sprintf("[red]Error: %v[-]  |  Press R to reload, c for settings", err))
				p.table.Clear()
				return
			}
			p.sessions = sessions
			p.renderTable()
			if len(sessions) == 0 {
				p.status.SetText("[gray]No sessions found. Is a player connected?  |  Press R to reload")
			} else {
				p.status.SetText(fmt.Sprintf("[green]%d session(s)[-]  |  Enter/l: connect  R: reload", len(sessions)))
			}
		})
	}()
}

func (p *SessionsPage) renderTable() {
	p.table.Clear()

	headers := []string{"Name", "Player", "Controllers", "ID"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		p.table.SetCell(0, i, cell)
	}

	for i, s := range p.sessions {
		row := i + 1

		playerStatus := "[red]offline[-]"
		if s.HasPlayer {
			playerStatus = "[green]online[-]"
		}

		name := s.Name
		if name == "" {
			name = "(unnamed)"
		}

		p.table.SetCell(row, 0, tview.NewTableCell(name).SetExpansion(1))
		p.table.SetCell(row, 1, tview.NewTableCell(playerStatus).SetExpansion(1))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", s.ControllerCount)).SetExpansion(1))
		p.table.SetCell(row, 3, tview.NewTableCell(s.ID).SetExpansion(1).SetTextColor(tcell.ColorGray))
	}

	if len(p.sessions) > 0 {
		p.table.Select(1, 0)
	}
}
