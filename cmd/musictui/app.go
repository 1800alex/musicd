package main

import (
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Page is implemented by all TUI pages.
type Page interface {
	tview.Primitive
	Load()
}

// App is the main application shell.
type App struct {
	tviewApp  *tview.Application
	pages     *tview.Pages
	statusBar *StatusBar
	tabBar    *tview.TextView
	layout    *tview.Flex

	client   *APIClient
	ws       *WSClient
	state    *PlayerState
	stateMu  sync.RWMutex
	pageSize int

	history     []string
	currentPage string

	// Registered pages for Load() calls
	pageMap map[string]Page

	// Session info
	sessionID   string
	sessionName string
}

// NewApp creates a new application.
func NewApp(serverURL string, pageSize int) *App {
	a := &App{
		tviewApp: tview.NewApplication(),
		pages:    tview.NewPages(),
		pageSize: pageSize,
		client:   NewAPIClient(serverURL),
		pageMap:  make(map[string]Page),
	}

	a.statusBar = NewStatusBar()

	a.tabBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.tabBar.SetBackgroundColor(tcell.ColorDarkBlue)
	a.updateTabBar()

	a.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.tabBar, 1, 0, false).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.statusBar, 2, 0, false)

	a.setupGlobalKeys()
	a.tviewApp.SetRoot(a.layout, true)

	return a
}

func (a *App) updateTabBar() {
	tabs := []struct{ key, name, page string }{
		{"1", "Tracks", "tracks"},
		{"2", "Artists", "artists"},
		{"3", "Playlists", "playlists"},
		{"q", "Queue", "queue"},
		{"c", "Connect", "connect"},
	}

	text := ""
	for i, tab := range tabs {
		if i > 0 {
			text += "  "
		}
		if a.currentPage == tab.page {
			text += "[black:white] " + tab.key + ":" + tab.name + " [-:-]"
		} else {
			text += "[white:darkblue] " + tab.key + ":" + tab.name + " [-:-]"
		}
	}
	a.tabBar.SetText(text)
}

// RegisterPage adds a page to the application.
func (a *App) RegisterPage(name string, page Page) {
	a.pages.AddPage(name, page, true, false)
	a.pageMap[name] = page
}

// NavigateTo switches to a page and pushes the current page to history.
func (a *App) NavigateTo(name string) {
	if a.currentPage != "" && a.currentPage != name {
		a.history = append(a.history, a.currentPage)
	}
	a.currentPage = name
	a.pages.SwitchToPage(name)
	a.updateTabBar()

	if page, ok := a.pageMap[name]; ok {
		page.Load()
	}
}

// GoBack pops the navigation history and switches to the previous page.
func (a *App) GoBack() {
	if len(a.history) == 0 {
		return
	}
	prev := a.history[len(a.history)-1]
	a.history = a.history[:len(a.history)-1]
	a.currentPage = prev
	a.pages.SwitchToPage(prev)
	a.updateTabBar()

	if page, ok := a.pageMap[prev]; ok {
		page.Load()
	}
}

// SendCommand sends a command to the connected player session.
func (a *App) SendCommand(action string, value interface{}) {
	if a.ws != nil {
		go a.ws.SendCommand(action, value)
	}
}

// ConnectToSession connects the WebSocket to a player session.
func (a *App) ConnectToSession(id, name string) {
	if a.ws != nil {
		a.ws.Close()
	}
	a.sessionID = id
	a.sessionName = name

	a.ws = NewWSClient(a.client.BaseURL, id, func(state PlayerState) {
		a.onStateUpdate(state)
	}, nil)
	go a.ws.Connect()
}

// DisconnectSession closes the WebSocket connection.
func (a *App) DisconnectSession() {
	if a.ws != nil {
		a.ws.Close()
		a.ws = nil
	}
	a.sessionID = ""
	a.sessionName = ""
	a.stateMu.Lock()
	a.state = nil
	a.stateMu.Unlock()
	a.tviewApp.QueueUpdateDraw(func() {
		a.statusBar.Update(nil)
	})
}

func (a *App) onStateUpdate(state PlayerState) {
	a.stateMu.Lock()
	a.state = &state
	a.stateMu.Unlock()

	a.tviewApp.QueueUpdateDraw(func() {
		a.statusBar.Update(&state)

		// Update queue page if visible
		if a.currentPage == "queue" {
			if page, ok := a.pageMap["queue"]; ok {
				page.Load()
			}
		}
	})
}

func (a *App) isInputFocused() bool {
	_, ok := a.tviewApp.GetFocus().(*tview.InputField)
	return ok
}

func (a *App) setupGlobalKeys() {
	a.tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Don't intercept when typing in an input field
		if a.isInputFocused() {
			return event
		}

		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case ' ':
				a.SendCommand("toggle_play", nil)
				return nil
			case 'n':
				a.SendCommand("next", nil)
				return nil
			case 'N':
				a.SendCommand("previous", nil)
				return nil
			case '+', '=':
				a.stateMu.RLock()
				vol := 50.0
				if a.state != nil {
					vol = a.state.Volume
				}
				a.stateMu.RUnlock()
				newVol := vol + 5
				if newVol > 100 {
					newVol = 100
				}
				a.SendCommand("volume", newVol)
				return nil
			case '-':
				a.stateMu.RLock()
				vol := 50.0
				if a.state != nil {
					vol = a.state.Volume
				}
				a.stateMu.RUnlock()
				newVol := vol - 5
				if newVol < 0 {
					newVol = 0
				}
				a.SendCommand("volume", newVol)
				return nil
			case 'm':
				a.SendCommand("toggle_mute", nil)
				return nil
			case 's':
				a.stateMu.RLock()
				shuffle := false
				if a.state != nil {
					shuffle = a.state.Shuffle
				}
				a.stateMu.RUnlock()
				a.SendCommand("set_shuffle", !shuffle)
				return nil
			case 'r':
				a.stateMu.RLock()
				mode := "Off"
				if a.state != nil {
					mode = a.state.RepeatMode
				}
				a.stateMu.RUnlock()
				nextMode := "Off"
				switch mode {
				case "Off":
					nextMode = "All"
				case "All":
					nextMode = "One"
				case "One":
					nextMode = "Off"
				}
				a.SendCommand("set_repeat", nextMode)
				return nil
			case 'q':
				a.NavigateTo("queue")
				return nil
			case '1':
				a.NavigateTo("tracks")
				return nil
			case '2':
				a.NavigateTo("artists")
				return nil
			case '3':
				a.NavigateTo("playlists")
				return nil
			case 'c':
				a.NavigateTo("connect")
				return nil
			case '?':
				a.showHelp()
				return nil
			}
		}

		return event
	})
}

func (a *App) showHelp() {
	modal := tview.NewModal().
		SetText(`[yellow]Keyboard Shortcuts[-]

[white]Navigation[-]
  1/2/3    Tracks/Artists/Playlists
  q        Queue
  c        Connect/Settings
  Enter/l  Select item
  h/Esc    Go back
  [/]      Prev/Next page
  /        Search

[white]Playback[-]
  Space    Play/Pause
  n        Next track
  N        Previous track
  +/-      Volume up/down
  m        Mute
  s        Shuffle
  r        Repeat (Off/All/One)
  a        Play all (on detail pages)

  ?        This help
  Ctrl+C   Quit`).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("help")
		})
	a.pages.AddPage("help", modal, true, true)
}

// Run starts the TUI application.
func (a *App) Run() error {
	return a.tviewApp.Run()
}

// GetState returns a copy of the current player state.
func (a *App) GetState() *PlayerState {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.state
}
