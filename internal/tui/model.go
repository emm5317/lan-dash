package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
)

type Model struct {
	store     *store.Store
	devices   []store.Device
	cursor    int
	filter    string
	filtering bool
	width     int
	height    int
}

func newModel(s *store.Store) tea.Model {
	return Model{
		store:   s,
		devices: s.All(),
	}
}

func (m Model) Init() tea.Cmd {
	return listenForUpdates(m.store)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateMsg:
		m.devices = m.store.All()
		return m, listenForUpdates(m.store)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		key := msg.String()

		// ctrl+c always quits
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		if m.filtering {
			switch key {
			case "esc":
				m.filter = ""
				m.filtering = false
			case "enter":
				m.filtering = false
			case "backspace":
				runes := []rune(m.filter)
				if len(runes) > 0 {
					m.filter = string(runes[:len(runes)-1])
				}
			default:
				if len(key) == 1 && key[0] >= 32 {
					m.filter += key
				}
			}
			m.cursor = 0
			return m, nil
		}

		// Normal mode
		switch key {
		case "/":
			m.filtering = true
		case "j", "down":
			if m.cursor < len(m.filtered())-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "s":
			return m, triggerScan(m.store)
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) filtered() []store.Device {
	if m.filter == "" {
		return m.devices
	}
	lower := strings.ToLower(m.filter)
	var out []store.Device
	for _, d := range m.devices {
		if strings.Contains(strings.ToLower(d.IP), lower) ||
			strings.Contains(strings.ToLower(d.Vendor), lower) ||
			strings.Contains(strings.ToLower(d.Hostname), lower) {
			out = append(out, d)
		}
	}
	return out
}

type updateMsg struct{}

func listenForUpdates(s *store.Store) tea.Cmd {
	return func() tea.Msg {
		ch, unsub := s.Subscribe()
		defer unsub()
		for range ch {
			return updateMsg{}
		}
		return nil
	}
}

func triggerScan(s *store.Store) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		scanner.TriggerScan(ctx, s)
		return nil
	}
}
