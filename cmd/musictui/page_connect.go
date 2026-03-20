package main

import (
	"fmt"

	"github.com/rivo/tview"
)

// ConnectPage allows editing the server URL and connection settings.
type ConnectPage struct {
	*tview.Flex
	form   *tview.Form
	status *tview.TextView
	app    *App
}

// NewConnectPage creates a new settings/connect page.
func NewConnectPage(app *App) *ConnectPage {
	p := &ConnectPage{
		app: app,
	}

	p.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	p.form = tview.NewForm().
		AddInputField("Server URL", app.client.BaseURL, 60, nil, nil).
		AddButton("Connect", func() {
			url := p.form.GetFormItemByLabel("Server URL").(*tview.InputField).GetText()
			app.client.BaseURL = url
			app.DisconnectSession()
			app.NavigateTo("sessions")
		}).
		AddButton("Back", func() {
			app.GoBack()
		})
	p.form.SetBorder(true).SetTitle(" Connection Settings ")

	p.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(p.form, 70, 0, true).
			AddItem(nil, 0, 1, false),
			10, 0, true).
		AddItem(p.status, 1, 0, false).
		AddItem(nil, 0, 1, false)

	return p
}

// Load updates the status display.
func (p *ConnectPage) Load() {
	if p.app.sessionID != "" {
		name := p.app.sessionName
		if name == "" {
			name = p.app.sessionID
		}
		p.status.SetText(fmt.Sprintf("[green]Connected to: %s[-]", name))
	} else {
		p.status.SetText("[gray]Not connected to any session")
	}

	// Update the URL field to current value
	if item := p.form.GetFormItemByLabel("Server URL"); item != nil {
		item.(*tview.InputField).SetText(p.app.client.BaseURL)
	}
}
