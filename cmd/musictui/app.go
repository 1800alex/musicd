package main

import (
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Page is implemented by all TUI pages.
type Page interface {
	tview.Primitive
	Load()
}

type InterpolatedProgress struct {
	Time       float64   // current interpolated time in seconds
	Dur        float64   // total duration
	Playing    bool      // whether playback is active
	LastUpd    time.Time // when we last corrected from server
	Mu         sync.Mutex
	tickerStop chan struct{}
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

	// Interpolated progress tracking
	progress InterpolatedProgress

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

	a.progress.tickerStop = make(chan struct{})
	go a.progressTicker()

	return a
}

func (a *App) updateTabBar() {
	tabs := []struct{ key, name, page string }{
		{"1", "Tracks", "tracks"},
		{"2", "Artists", "artists"},
		{"3", "Playlists", "playlists"},
		{"p", "Playing", "nowplaying"},
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

	a.tviewApp.SetFocus(a.pages)
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

	// Clean up dynamic detail pages when leaving them
	if a.currentPage == "artist-detail" || a.currentPage == "album-detail" || a.currentPage == "playlist-detail" {
		a.pages.RemovePage(a.currentPage)
		delete(a.pageMap, a.currentPage)
	}

	a.currentPage = prev
	a.pages.SwitchToPage(prev)
	a.updateTabBar()

	a.tviewApp.SetFocus(a.pages)
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
		a.statusBar.Update(nil, 0)
	})
}

func (a *App) onStateUpdate(state PlayerState) {
	a.stateMu.Lock()
	prevTrackID := ""
	if a.state != nil && a.state.CurrentTrack != nil {
		prevTrackID = a.state.CurrentTrack.ID
	}
	a.state = &state
	a.stateMu.Unlock()

	newTrackID := ""
	if state.CurrentTrack != nil {
		newTrackID = state.CurrentTrack.ID
	}
	trackChanged := prevTrackID != newTrackID

	// Correct interpolated progress from server state
	a.progress.Mu.Lock()
	a.progress.Time = state.CurrentTime
	a.progress.Dur = state.Duration
	a.progress.Playing = state.IsPlaying
	a.progress.LastUpd = time.Now()
	a.progress.Mu.Unlock()

	a.tviewApp.QueueUpdateDraw(func() {
		a.statusBar.Update(&state, state.CurrentTime)

		// Update now playing page if visible
		if a.currentPage == "nowplaying" {
			if page, ok := a.pageMap["nowplaying"]; ok {
				page.Load()
			}
		}
		// Re-render track tables when current track changes,
		// preserving the user's current selection position.
		if trackChanged {
			switch a.currentPage {
			case "tracks":
				if p, ok := a.pageMap["tracks"].(*TracksPage); ok {
					row, _ := p.table.GetSelection()
					p.renderTable()
					if row > 0 && row < p.table.GetRowCount() {
						p.table.Select(row, 0)
					}
				}
			case "artist-detail":
				if p, ok := a.pageMap["artist-detail"].(*ArtistDetailPage); ok {
					row, _ := p.trackTable.GetSelection()
					p.renderTrackTable()
					if row > 0 && row < p.trackTable.GetRowCount() {
						p.trackTable.Select(row, 0)
					}
				}
			case "playlist-detail":
				if p, ok := a.pageMap["playlist-detail"].(*PlaylistDetailPage); ok {
					row, _ := p.table.GetSelection()
					p.renderTable()
					if row > 0 && row < p.table.GetRowCount() {
						p.table.Select(row, 0)
					}
				}
			}
		}
	})
}

// seekRelative seeks forward or backward by the given number of seconds.
func (a *App) seekRelative(delta float64) {
	a.progress.Mu.Lock()
	newPos := a.progress.Time + delta
	if newPos < 0 {
		newPos = 0
	}
	if a.progress.Dur > 0 && newPos > a.progress.Dur {
		newPos = a.progress.Dur
	}
	a.progress.Time = newPos
	a.progress.LastUpd = time.Now()
	a.progress.Mu.Unlock()

	a.SendCommand("seek", newPos)
}

// GetInterpolatedProgress returns the interpolated playback position and duration.
func (a *App) GetInterpolatedProgress() (currentTime, duration float64) {
	a.progress.Mu.Lock()
	cur := a.progress.Time
	dur := a.progress.Dur
	playing := a.progress.Playing
	lastUpd := a.progress.LastUpd
	a.progress.Mu.Unlock()

	if playing && dur > 0 {
		cur += time.Since(lastUpd).Seconds()
		if cur > dur {
			cur = dur
		}
	}
	return cur, dur
}

// progressTicker runs a 100ms ticker that interpolates the current playback
// position between server state updates, keeping the progress bar smooth.
func (a *App) progressTicker() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.progress.Mu.Lock()
			playing := a.progress.Playing
			dur := a.progress.Dur
			a.progress.Mu.Unlock()

			// Only queue redraws when actually playing
			if !playing || dur <= 0 {
				continue
			}

			interpolated, _ := a.GetInterpolatedProgress()

			a.tviewApp.QueueUpdateDraw(func() {
				a.stateMu.RLock()
				st := a.state
				a.stateMu.RUnlock()
				if st != nil {
					a.statusBar.Update(st, interpolated)
				}
				// Lightweight update for now playing page time only
				if a.currentPage == "nowplaying" {
					if np, ok := a.pageMap["nowplaying"].(*NowPlayingPage); ok {
						np.UpdateTime()
					}
				}
			})
		case <-a.progress.tickerStop:
			return
		}
	}
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
			case 'f':
				a.seekRelative(10)
				return nil
			case 'F':
				a.seekRelative(30)
				return nil
			case 'b':
				a.seekRelative(-10)
				return nil
			case 'B':
				a.seekRelative(-30)
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
			case 'c':
				a.NavigateTo("connect")
				return nil
			case '1', '2', '3', 'p':
				// Require an active session for all pages except connect
				if a.sessionID == "" {
					return nil
				}
				switch event.Rune() {
				case '1':
					a.NavigateTo("tracks")
				case '2':
					a.NavigateTo("artists")
				case '3':
					a.NavigateTo("playlists")
				case 'p':
					a.NavigateTo("nowplaying")
				}
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
	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetText(`
 [yellow::b]Keyboard Shortcuts[-:-:-]

 [white::b]Navigation[-:-:-]
 1/2/3      Tracks / Artists / Playlists
 p          Now Playing
 c          Connect / Settings
 Enter/l    Select item
 h/Bksp     Go back
 Esc        Go back / Clear search
 [[]]/←→    Prev / Next page
 /          Search
 g/G        First / Last row

 [white::b]Playback[-:-:-]
 Space      Play / Pause
 n/N        Next / Previous track
 ←/→        Next / Prev (Now Playing)
 f/b        Seek ±10s
 F/B        Seek ±30s
 +/-        Volume up / down
 m          Mute
 s          Shuffle
 r          Repeat (Off/All/One)
 a          Play all (detail pages)

 ?          This help
 Ctrl+C     Quit

 [gray]Press Esc or ? to close[-]`)
	helpText.SetBorder(true).SetTitle(" Help ")
	helpText.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			a.pages.RemovePage("help")
			return nil
		case tcell.KeyRune:
			if event.Rune() == '?' {
				a.pages.RemovePage("help")
				return nil
			}
		}
		return event
	})

	// Center the help text in a fixed-size box
	helpBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(helpText, 46, 0, true).
			AddItem(nil, 0, 1, false),
			30, 0, true).
		AddItem(nil, 0, 1, false)

	a.pages.AddPage("help", helpBox, true, true)
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
