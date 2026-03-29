package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
)

type Model struct {
	store   *store.Store
	devices []store.Device
	cursor  int
	filter  string
	width   int
	height  int
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
		switch msg.String() {
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
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	filtered := m.filtered()

	// Header with filter info
	header := "LAN Dashboard"
	if m.filter != "" {
		header += " (filtered: " + m.filter + ")"
	}
	header += "\n\n"

	// Controls
	controls := "j/k: navigate • s: scan • q: quit\n\n"

	// Table
	table := renderTable(filtered)

	// Cursor indicator
	if len(filtered) > 0 && m.cursor < len(filtered) {
		cursorLine := fmt.Sprintf("\n→ %s", filtered[m.cursor].IP)
		table += cursorLine
	}

	return header + controls + table
}

func (m Model) filtered() []store.Device {
	if m.filter == "" {
		return m.devices
	}
	var out []store.Device
	for _, d := range m.devices {
		if strings.Contains(d.IP, m.filter) ||
			strings.Contains(d.Vendor, m.filter) ||
			strings.Contains(d.Hostname, m.filter) {
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
