package main

import (
	"context"
	"log/slog"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/emm5317/lan-dash/internal/api"
	"github.com/emm5317/lan-dash/internal/history"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
	"github.com/emm5317/lan-dash/internal/tui"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s := store.New()

	// SQLite scan history
	db, err := history.Open("lan-dash.db")
	if err != nil {
		slog.Error("failed to open history db", "err", err)
		return
	}
	defer db.Close()
	go db.Listen(ctx, s)

	// Sweep subnet to populate ARP table, then discover mDNS hostnames
	scanner.SweepSubnet(ctx)
	scanner.DiscoverMDNS(ctx, s, 3*time.Second)

	h := api.NewHandler(s, db)
	t := tui.NewTUI(s)

	// Scanner
	go func() {
		if err := scanner.Run(ctx, s); err != nil {
			slog.Info("scanner stopped", "reason", err)
		}
	}()

	// HTTP server
	httpSrv := &http.Server{Addr: ":3000", Handler: h.Handler()}

	// SSH server
	sshSrv := t.NewSSHServer()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		slog.Info("HTTP listening", "addr", ":3000")
		httpSrv.ListenAndServe()
	}()

	go func() {
		defer wg.Done()
		slog.Info("SSH listening", "addr", ":2223")
		sshSrv.ListenAndServe()
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	httpSrv.Shutdown(shutdownCtx)
	sshSrv.Shutdown(shutdownCtx)
	wg.Wait()
}
