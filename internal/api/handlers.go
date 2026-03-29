package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"time"

	"github.com/emm5317/lan-dash/internal/history"
	"github.com/emm5317/lan-dash/internal/nickname"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
)

type Handler struct {
	store     *store.Store
	db        *history.DB
	nicknames *nickname.Store
}

func NewHandler(s *store.Store, db *history.DB, nicks *nickname.Store) *Handler {
	return &Handler{store: s, db: db, nicknames: nicks}
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
	mux.HandleFunc("/api/events", DatastarHandler(h.store, h.nicknames))
	mux.HandleFunc("/api/scan", h.scanDevices)
	mux.HandleFunc("/api/history", h.getHistory)
	mux.HandleFunc("/api/history/all", h.getAllHistory)
	mux.HandleFunc("/api/nickname", h.setNickname)
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	return mux
}

// deviceResponse is the JSON shape returned by /api/devices.
type deviceResponse struct {
	IP        string  `json:"ip"`
	MAC       string  `json:"mac"`
	Vendor    string  `json:"vendor"`
	Hostname  string  `json:"hostname"`
	Nickname  string  `json:"nickname"`
	RTTms     float64 `json:"rtt_ms"`
	OpenPorts []int   `json:"open_ports"`
	Alive     bool    `json:"alive"`
	Group     string  `json:"group"`
	Speed     string  `json:"speed"`
}

func (h *Handler) getDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	devices := h.store.All()
	out := make([]deviceResponse, 0, len(devices))
	for _, d := range devices {
		out = append(out, deviceResponse{
			IP:        d.IP,
			MAC:       d.MAC,
			Vendor:    d.Vendor,
			Hostname:  d.Hostname,
			Nickname:  h.nicknames.Get(d.IP),
			RTTms:     d.RTTms(),
			OpenPorts: d.OpenPorts,
			Alive:     d.Alive,
			Group:     string(d.Group()),
			Speed:     d.Speed,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
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

func (h *Handler) getAllHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	devices := h.store.All()
	result := make(map[string][]history.Snapshot)
	for _, d := range devices {
		snaps, _ := h.db.History(d.IP, 24*time.Hour)
		if len(snaps) > 0 {
			result[d.IP] = snaps
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// setNickname handles POST /api/nickname with JSON body {"ip":"...","nickname":"..."}.
func (h *Handler) setNickname(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		IP       string `json:"ip"`
		Nickname string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.IP == "" {
		http.Error(w, "ip required", http.StatusBadRequest)
		return
	}
	if err := h.nicknames.Set(body.IP, body.Nickname); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
