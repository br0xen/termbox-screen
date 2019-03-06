package termboxScreen

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"syscall"
	"time"

	termbox "github.com/nsf/termbox-go"
)

const (
	NoRefresh = 0
)

type Screen interface {
	Id() int
	Initialize(Bundle) error
	HandleKeyEvent(termbox.Event) int
	HandleNoneEvent(termbox.Event) int
	DrawScreen()
	ResizeScreen()
}

type Manager struct {
	defaultFg termbox.Attribute
	defaultBg termbox.Attribute

	screens         map[int]Screen
	displayScreenId int
	events          chan termbox.Event

	running     bool
	refreshRate time.Duration
}

func NewManager() *Manager {
	m := Manager{
		defaultFg:   termbox.ColorWhite,
		defaultBg:   termbox.ColorBlack,
		events:      make(chan termbox.Event),
		screens:     make(map[int]Screen),
		refreshRate: NoRefresh,
	}
	if err := termbox.Init(); err != nil {
		fmt.Println("Error initializing termbox")
		return nil
	}
	termbox.SetOutputMode(termbox.Output256)
	return &m
}

func (m *Manager) SetDefaultFg(c termbox.Attribute) { m.defaultFg = c }
func (m *Manager) SetDefaultBg(c termbox.Attribute) { m.defaultBg = c }

// AddScreen adds a screen to the screens map, they're indexed by the value
// returned from the screen's Id() function
func (m *Manager) AddScreen(s Screen) {
	m.screens[s.Id()] = s
	// If this is the only screen we've added, set it to active
	if len(m.screens) == 1 {
		m.SetDisplayScreen(s.Id())
	}
}

func (m *Manager) GetScreens() map[int]Screen {
	return m.screens
}

func (m *Manager) SetDisplayScreen(id int) error {
	if id == m.displayScreenId {
		return nil
	}
	var ok bool
	if _, ok = m.screens[id]; !ok {
		return errors.New("Invalid Screen Id")
	}
	m.displayScreenId = id
	return nil
}

func (m *Manager) InitializeScreen(id int, b Bundle) error {
	var ok bool
	if _, ok = m.screens[id]; !ok {
		return errors.New("Invalid screen id")
	}
	return m.screens[id].Initialize(b)
}

func (m *Manager) Loop() error {
	if len(m.screens) == 0 {
		return errors.New("Loop cannot run without screens")
	}
	m.running = true
	go m.pollUserEvents()
	// We always start display the first screen added
	m.layoutAndDrawScreen()
	for {
		event := <-m.events
		if event.Type == termbox.EventKey {
			if event.Key == termbox.KeyCtrlC {
				break
			} else if event.Key == termbox.KeyCtrlZ {
				if runtime.GOOS != "windows" {
					process, _ := os.FindProcess(os.Getpid())
					termbox.Close()
					process.Signal(syscall.SIGSTOP)
					termbox.Init()
				}
			} else {
				newScreenIndex := m.handleKeyEvent(event)
				if err := m.SetDisplayScreen(newScreenIndex); err != nil {
					break
				}
				m.layoutAndDrawScreen()
			}
		} else if event.Type == termbox.EventNone {
			// Type = EventNone is how we can trigger automatic events
			newScreenIndex := m.handleNoneEvent(event)
			if err := m.SetDisplayScreen(newScreenIndex); err != nil {
				break
			}
			m.layoutAndDrawScreen()
		} else if event.Type == termbox.EventResize {
			m.resizeScreen()
			m.layoutAndDrawScreen()
		}
	}
	m.running = false
	close(m.events)
	termbox.Close()
	return nil
}

func (m *Manager) SendNoneEvent() {
	m.SendEvent(termbox.Event{Type: termbox.EventNone})
}

func (m *Manager) SendEvent(t termbox.Event) {
	m.events <- t
}

func (m *Manager) pollUserEvents() {
	for m.running {
		m.SendEvent(termbox.PollEvent())
	}
}

func (m *Manager) SetRefreshRate(t time.Duration) {
	m.refreshRate = t
	go m.pollRefreshEvents()
}

func (m *Manager) pollRefreshEvents() {
	if m.refreshRate > time.Microsecond {
		for m.running {
			ioutil.WriteFile("./log", []byte(time.Now().Format(time.RFC3339)), 0644)
			time.Sleep(m.refreshRate)
			m.SendNoneEvent()
		}
	}
}

func (m *Manager) handleKeyEvent(event termbox.Event) int {
	return m.screens[m.displayScreenId].HandleKeyEvent(event)
}

func (m *Manager) handleNoneEvent(event termbox.Event) int {
	return m.screens[m.displayScreenId].HandleNoneEvent(event)
}

func (m *Manager) resizeScreen() {
	m.screens[m.displayScreenId].ResizeScreen()
}

func (m *Manager) drawBackground() {
	termbox.Clear(0, m.defaultBg)
}

func (m *Manager) layoutAndDrawScreen() {
	m.drawBackground()
	m.screens[m.displayScreenId].DrawScreen()
	termbox.Flush()
}
