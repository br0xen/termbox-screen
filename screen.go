package termboxScreen

import (
	"errors"
	"os"
	"runtime"
	"syscall"

	termbox "github.com/nsf/termbox-go"
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
}

func NewManager() *Manager {
	m := Manager{
		defaultFg: termbox.ColorWhite,
		defaultBg: termbox.ColorBlack,
		events:    make(chan termbox.Event),
		screens:   make(map[int]Screen),
	}
	// Add the default user-input provider
	m.AddEventProvider(func(e chan termbox.Event) {
		for {
			e <- termbox.PollEvent()
		}
	})
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

func (m *Manager) AddEventProvider(provider func(chan termbox.Event)) {
	go provider(m.events)
}

func (m *Manager) Loop() error {
	if len(m.screens) == 0 {
		return errors.New("Loop cannot run without screens")
	}
	if err := termbox.Init(); err != nil {
		return err
	}
	termbox.SetOutputMode(termbox.Output256)
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
	termbox.Close()
	return nil
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
