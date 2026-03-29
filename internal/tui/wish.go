package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	wishbt "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/emm5317/lan-dash/internal/store"
	"os"
)

type TUI struct {
	store *store.Store
}

func NewTUI(s *store.Store) *TUI {
	return &TUI{store: s}
}

func (t *TUI) StartSSH() {
	// Check if SSH host key exists, generate if missing
	keyPath := ".ssh/host_key"
	if stat, err := os.Stat(keyPath); os.IsNotExist(err) || (err == nil && stat.Size() == 0) {
		// Remove empty key file if it exists
		if err == nil && stat.Size() == 0 {
			os.Remove(keyPath)
		}
		// Create .ssh directory if it doesn't exist
		if err := os.MkdirAll(".ssh", 0700); err != nil {
			panic("failed to create .ssh directory: " + err.Error())
		}
		// Generate Ed25519 key and write to disk
		_, err := keygen.New(keyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
		if err != nil {
			panic("failed to generate SSH host key: " + err.Error())
		}
	}

	s, err := wish.NewServer(
		wish.WithAddress(":2223"),
		wish.WithHostKeyPath(keyPath),
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
