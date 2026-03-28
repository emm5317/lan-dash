package api

import (
	"encoding/json"
	"github.com/emm5317/lan-dash/internal/store"
	"net/http"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) StartHTTP() {
	http.HandleFunc("/api/devices", h.getDevices)
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
