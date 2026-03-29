package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/emm5317/lan-dash/internal/history"
	"github.com/emm5317/lan-dash/internal/nickname"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
)

type Model struct {
	store            *store.Store
	nicknames        *nickname.Store
	db               *history.DB
	devices          []store.Device
	sparklines       map[string]string
	lastSparkRefresh time.Time
	cursor           int
	filter           string
	filtering        bool
	editing          bool
	editInput        string
	width            int
	height            int
}

func newModel(s *store.Store, nicks *nickname.Store, db *history.DB) tea.Model {
	return Model{
		store:      s,
		nicknames:  nicks,
		db:         db,
		devices:    s.All(),
		sparklines: make(map[string]string),
	}
}

func (m Model) Init() tea.Cmd {
	return listenForUpdates(m.store)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateMsg:
		m.devices = m.store.All()
		if m.db != nil && time.Since(m.lastSparkRefresh) >= 30*time.Second {
			m.lastSparkRefresh = time.Now()
			for _, d := range m.devices {
				snaps, err := m.db.History(d.IP, 1*time.Hour)
				if err == nil && len(snaps) > 0 {
					rtts := make([]float64, len(snaps))
					for i, s := range snaps {
						rtts[i] = s.RTTms
					}
					m.sparklines[d.IP] = sparkline(rtts, 20)
				}
			}
		}
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

		// Editing mode: set nickname for selected device
		if m.editing {
			switch key {
			case "esc":
				m.editing = false
				m.editInput = ""
			case "enter":
				devs := m.filtered()
				if m.cursor < len(devs) && m.nicknames != nil {
					ip := devs[m.cursor].IP
					m.nicknames.Set(ip, m.editInput)
				}
				m.editing = false
				m.editInput = ""
			case "backspace":
				runes := []rune(m.editInput)
				if len(runes) > 0 {
					m.editInput = string(runes[:len(runes)-1])
				}
			default:
				if len(key) == 1 && key[0] >= 32 {
					m.editInput += key
				}
			}
			return m, nil
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
		case "n":
			devs := m.filtered()
			if m.cursor < len(devs) {
				// Pre-fill with existing nickname
				if m.nicknames != nil {
					m.editInput = m.nicknames.Get(devs[m.cursor].IP)
				}
				m.editing = true
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
		nick := ""
		if m.nicknames != nil {
			nick = m.nicknames.Get(d.IP)
		}
		if strings.Contains(strings.ToLower(d.IP), lower) ||
			strings.Contains(strings.ToLower(d.Vendor), lower) ||
			strings.Contains(strings.ToLower(d.Hostname), lower) ||
			strings.Contains(strings.ToLower(nick), lower) {
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
