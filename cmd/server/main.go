package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/emm5317/lan-dash/internal/api"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
	"github.com/emm5317/lan-dash/internal/tui"
)

func main() {
	s := store.New()
	h := api.NewHandler(s)
	t := tui.NewTUI(s)

	ctx := context.Background()

	// Start scanner
	go func() {
		if err := scanner.Run(ctx, s); err != nil {
			slog.Error("scanner failed", "err", err)
		}
	}()

	// Start HTTP server
	go h.StartHTTP()

	// Start SSH TUI server
	t.StartSSH()

	log.Println("LAN Dash started")
	select {} // Block forever
}
