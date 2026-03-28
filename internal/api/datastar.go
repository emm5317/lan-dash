package api

import (
	"fmt"
	"github.com/emm5317/lan-dash/internal/store"
	"net/http"
)

func (h *Handler) sseEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for event := range h.store.Events() {
		// Datastar fragment for device update
		fragment := fmt.Sprintf(`<tr id="device-%s"><td>%s</td><td>%s</td><td>%v</td></tr>`,
			event.Device.IP, event.Device.IP, event.Device.MAC, event.Device.RTT)
		fmt.Fprintf(w, "data: %s\n\n", fragment)
		flusher.Flush()
	}
}
