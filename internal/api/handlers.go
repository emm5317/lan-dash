package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"time"

	"github.com/emm5317/lan-dash/internal/history"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
)

type Handler struct {
	store *store.Store
	db    *history.DB
}

func NewHandler(s *store.Store, db *history.DB) *Handler {
	return &Handler{store: s, db: db}
}

// getWebDir returns the path to the web directory.
// It first tries the executable's directory, then falls back to the current working directory.
// This handles both `go run` (where the executable is in a temp dir) and built binaries.
func getWebDir() string {
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		webDir := filepath.Join(execDir, "web")
		if _, err := os.Stat(webDir); err == nil {
			return webDir
		}
	}
	// Fall back to current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "web"
	}
	return filepath.Join(wd, "web")
}

func (h *Handler) Handler() http.Handler {
	webDir := getWebDir()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/devices", h.getDevices)
	mux.HandleFunc("/api/events", DatastarHandler(h.store))
	mux.HandleFunc("/api/scan", h.scanDevices)
	mux.HandleFunc("/api/history", h.getHistory)
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	return mux
}

func (h *Handler) getDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	devices := h.store.All()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (h *Handler) scanDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Trigger enrichment for all known devices
	ctx := context.Background()
	scanner.TriggerScan(ctx, h.store)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Scan initiated"))
}

func (h *Handler) getHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "ip parameter required", http.StatusBadRequest)
		return
	}
	snapshots, err := h.db.History(ip, 24*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshots)
}
