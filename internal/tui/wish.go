package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	wishbt "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/emm5317/lan-dash/internal/store"
)

type TUI struct {
	store *store.Store
}

func NewTUI(s *store.Store) *TUI {
	return &TUI{store: s}
}

func (t *TUI) StartSSH() {
	s, err := wish.NewServer(
		wish.WithAddress(":2222"),
		wish.WithHostKeyPath(".ssh/host_key"),
		wish.WithMiddleware(
			wishbt.Middleware(t.teaHandler),
			logging.Middleware(),
		),
	)
	if err != nil {
		panic(err)
	}
	s.ListenAndServe()
}

func (t *TUI) teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return newModel(t.store), []tea.ProgramOption{tea.WithAltScreen()}
}
