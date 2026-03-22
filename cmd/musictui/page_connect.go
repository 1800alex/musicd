package main

import (
	"fmt"
	"musicd/lib/types"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ConnectPage allows editing the server URL and browsing/selecting sessions.
type ConnectPage struct {
	*tview.Flex
	form     *tview.Form
	urlInput *tview.InputField
	table    *tview.Table
	status   *tview.TextView
	app      *App
	sessions []types.SessionInfo
}

// NewConnectPage creates a new connect page with URL input and session list.
func NewConnectPage(app *App) *ConnectPage {
	p := &ConnectPage{
		app: app,
	}

	p.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.urlInput = tview.NewInputField().
		SetLabel("Server URL ").
		SetText(app.client.BaseURL).
		SetFieldWidth(60)

	p.form = tview.NewForm()
	p.form.AddFormItem(p.urlInput)
	p.form.AddButton("Fetch Sessions", func() {
		p.fetchSessions()
	})
	p.form.SetBorder(true).SetTitle(" Connection Settings ")

	p.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	p.table.SetBorder(true).SetTitle(" Sessions ")

	p.table.SetSelectedFunc(func(row, col int) {
		p.selectSession(row)
	})

	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			app.tviewApp.SetFocus(p.urlInput)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'l':
				row, _ := p.table.GetSelection()
				p.selectSession(row)
				return nil
			case 'R':
				p.fetchSessions()
				return nil
			}
		}
		return event
	})

	p.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if len(p.sessions) > 0 {
				app.tviewApp.SetFocus(p.table)
			}
			return nil
		}
		return event
	})

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.form, 7, 0, true).
		AddItem(p.table, 0, 1, false).
		AddItem(p.status, 1, 0, false)

	return p
}

func (p *ConnectPage) selectSession(row int) {
	idx := row - 1 // skip header
	if idx >= 0 && idx < len(p.sessions) {
		s := p.sessions[idx]
		p.app.ConnectToSession(s.ID, s.Name)
		p.app.NavigateTo("nowplaying")
	}
}

func (p *ConnectPage) fetchSessions() {
	// Apply the URL from the form and persist it
	p.app.client.BaseURL = p.urlInput.GetText()
	SaveConfig(&Config{ServerURL: p.app.client.BaseURL})

	p.status.SetText("[yellow]Loading sessions...")
	go func() {
		sessions, err := p.app.client.GetSessions()
		p.app.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				p.status.SetText(fmt.Sprintf("[red]Error: %v[-]  |  Press R to reload", err))
				p.sessions = nil
				p.table.Clear()
				return
			}
			p.sessions = sessions
			p.renderTable()
			if len(sessions) == 0 {
				p.status.SetText("[gray]No sessions found. Is a player connected?  |  Press R to reload")
			} else {
				p.status.SetText(fmt.Sprintf("[green]%d session(s)[-]  |  Enter/l: connect  R: reload  Tab: switch focus", len(sessions)))
				p.app.tviewApp.SetFocus(p.table)
			}
		})
	}()
}

func (p *ConnectPage) renderTable() {
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

// Load updates the status display and fetches sessions.
func (p *ConnectPage) Load() {
	// Update the URL field to current value
	p.urlInput.SetText(p.app.client.BaseURL)

	if p.app.sessionID != "" {
		name := p.app.sessionName
		if name == "" {
			name = p.app.sessionID
		}
		p.status.SetText(fmt.Sprintf("[green]Connected to: %s[-]  |  Tab: switch focus", name))
	} else {
		p.status.SetText("[gray]Not connected  |  Fetch sessions to connect")
	}

	// Focus the URL input so it's immediately editable
	p.app.tviewApp.SetFocus(p.urlInput)

	// Auto-fetch sessions on load
	p.fetchSessions()
}
