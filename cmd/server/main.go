package main

import (
	"log"

	"github.com/emm5317/lan-dash/internal/api"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
	"github.com/emm5317/lan-dash/internal/tui"
)

func main() {
	s := store.New()
	sc := scanner.NewScanner(s)
	h := api.NewHandler(s, sc)
	t := tui.NewTUI(s)

	// Start scanner in goroutine
	go sc.StartARP()

	// Start HTTP server
	go h.StartHTTP()

	// Start SSH TUI server
	t.StartSSH()

	log.Println("LAN Dash started")
	select {} // Block forever
}
