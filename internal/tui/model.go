package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/emm5317/lan-dash/internal/store"
)

type Model struct {
	store *store.Store
	table string
}

func newModel(s *store.Store) tea.Model {
	return Model{
		store: s,
		table: renderTable(s.All()),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		listenForUpdates(m.store),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case updateMsg:
		m.table = renderTable(m.store.All())
	}
	return m, nil
}

func (m Model) View() string {
	return m.table
}

type updateMsg struct{}

func listenForUpdates(s *store.Store) tea.Cmd {
	return func() tea.Msg {
		ch, _ := s.Subscribe()
		for range ch {
			return updateMsg{}
		}
		return nil
	}
}
