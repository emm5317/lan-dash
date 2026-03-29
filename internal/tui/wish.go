package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	wishbt "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/emm5317/lan-dash/internal/history"
	"github.com/emm5317/lan-dash/internal/nickname"
	"github.com/emm5317/lan-dash/internal/store"
)

type TUI struct {
	store     *store.Store
	nicknames *nickname.Store
	db        *history.DB
}

func NewTUI(s *store.Store, nicks *nickname.Store, db *history.DB) *TUI {
	return &TUI{store: s, nicknames: nicks, db: db}
}

func (t *TUI) NewSSHServer(port int) *ssh.Server {
	// Check if SSH host key exists, generate if missing
	keyPath := ".ssh/host_key"
	if stat, err := os.Stat(keyPath); os.IsNotExist(err) || (err == nil && stat.Size() == 0) {
		if err == nil && stat.Size() == 0 {
			os.Remove(keyPath)
		}
		if err := os.MkdirAll(".ssh", 0700); err != nil {
			panic("failed to create .ssh directory: " + err.Error())
		}
		_, err := keygen.New(keyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
		if err != nil {
			panic("failed to generate SSH host key: " + err.Error())
		}
	}

	addr := fmt.Sprintf(":%d", port)
	s, err := wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(keyPath),
		wish.WithMiddleware(
			wishbt.Middleware(t.teaHandler),
			logging.Middleware(),
		),
	)
	if err != nil {
		panic(err)
	}
	return s
}

func (t *TUI) teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return newModel(t.store, t.nicknames, t.db), []tea.ProgramOption{tea.WithAltScreen()}
}
