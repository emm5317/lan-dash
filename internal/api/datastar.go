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

	ch, unsubscribe := h.store.Subscribe()
	defer unsubscribe()

	for event := range ch {
		var fragment string
		if event.Type == store.EventUpsert {
			// Datastar fragment for device update
			fragment = fmt.Sprintf(`<tr id="%s"><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%.2f</td><td>%v</td></tr>`,
				event.Device.SafeID(), event.Device.IP, event.Device.MAC, event.Device.Vendor, event.Device.Hostname, event.Device.RTTms(), event.Device.Alive)
		} else if event.Type == store.EventOffline {
			fragment = fmt.Sprintf(`<tr id="%s" class="offline"><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%.2f</td><td>%v</td></tr>`,
				event.Device.SafeID(), event.Device.IP, event.Device.MAC, event.Device.Vendor, event.Device.Hostname, event.Device.RTTms(), event.Device.Alive)
		}
		fmt.Fprintf(w, "data: %s\n\n", fragment)
		flusher.Flush()
	}
}
