package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/emm5317/lan-dash/internal/api"
	"github.com/emm5317/lan-dash/internal/config"
	"github.com/emm5317/lan-dash/internal/history"
	"github.com/emm5317/lan-dash/internal/nickname"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
	"github.com/emm5317/lan-dash/internal/tui"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("config.json")
	if err != nil {
		slog.Error("failed to load config", "err", err)
		return
	}

	s := store.New()

	// SQLite scan history
	db, err := history.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open history db", "err", err)
		return
	}
	defer db.Close()
	go db.Listen(ctx, s)

	// Load nicknames
	nicks, err := nickname.Load(cfg.NicknamesPath)
	if err != nil {
		slog.Error("failed to load nicknames", "err", err)
		return
	}

	// Sweep subnet to populate ARP table, then discover mDNS hostnames
	scanner.SweepSubnet(ctx, cfg.SweepWorkers)
	scanner.DiscoverMDNS(ctx, s, 3*time.Second)

	h := api.NewHandler(s, db, nicks)
	t := tui.NewTUI(s, nicks, db)

	// Build scanner options from config
	opts := scanner.Options{
		Interval:    cfg.ScanInterval.Duration,
		PingTimeout: cfg.PingTimeout.Duration,
		PortTimeout: cfg.PortTimeout.Duration,
		Ports:       cfg.CommonPorts,
	}

	// Scanner
	go func() {
		if err := scanner.Run(ctx, s, opts); err != nil {
			slog.Info("scanner stopped", "reason", err)
		}
	}()

	// HTTP server
	httpAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpSrv := &http.Server{Addr: httpAddr, Handler: h.Handler()}

	// SSH server
	sshSrv := t.NewSSHServer(cfg.SSHPort)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		slog.Info("HTTP listening", "addr", httpAddr)
		httpSrv.ListenAndServe()
	}()

	go func() {
		defer wg.Done()
		slog.Info("SSH listening", "addr", fmt.Sprintf(":%d", cfg.SSHPort))
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
