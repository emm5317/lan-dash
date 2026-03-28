package api

import (
	"encoding/json"
	"github.com/emm5317/lan-dash/internal/scanner"
	"github.com/emm5317/lan-dash/internal/store"
	"net/http"
)

type Handler struct {
	store   *store.Store
	scanner *scanner.Scanner
}

func NewHandler(s *store.Store, sc *scanner.Scanner) *Handler {
	return &Handler{store: s, scanner: sc}
}

func (h *Handler) StartHTTP() {
	http.HandleFunc("/api/devices", h.getDevices)
	http.HandleFunc("/api/scan", h.scanDevice)
	http.HandleFunc("/api/events", h.sseEvents)
	http.ListenAndServe(":8080", nil)
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

func (h *Handler) scanDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "Missing ip parameter", http.StatusBadRequest)
		return
	}
	go h.scanner.ScanPorts(ip) // async
	w.WriteHeader(http.StatusAccepted)
}
