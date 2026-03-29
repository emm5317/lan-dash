package api

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/emm5317/lan-dash/internal/store"
)

func DatastarHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}

		// Send the full table immediately on connect — wrap in tbody for morph
		var buf bytes.Buffer
		buf.WriteString(`<tbody id="device-tbody">`)
		renderTable(&buf, s.All())
		buf.WriteString(`</tbody>`)
		writeFragment(w, "#device-tbody", "morph", buf.String())
		flusher.Flush()

		events, unsub := s.Subscribe()
		defer unsub()

		for {
			select {
			case ev := <-events:
				buf.Reset()
				renderRow(&buf, ev.Device)
				// morph merges by id — existing row updates in place
				writeFragment(w, "#"+ev.Device.SafeID(), "morph", buf.String())
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

// writeFragment emits a Datastar fragment SSE event.
func writeFragment(w io.Writer, selector, mergeMode, fragment string) {
	oneLine := strings.Join(strings.Fields(strings.TrimSpace(fragment)), " ")
	fmt.Fprintf(w, "event: datastar-fragment\n")
	fmt.Fprintf(w, "data: selector %s\n", selector)
	fmt.Fprintf(w, "data: merge %s\n", mergeMode)
	fmt.Fprintf(w, "data: settle 0\n")
	fmt.Fprintf(w, "data: fragment %s\n\n", oneLine)
}

func renderTable(buf *bytes.Buffer, devices []store.Device) {
	for _, d := range devices {
		renderRow(buf, d)
	}
}

func renderRow(buf *bytes.Buffer, d store.Device) {
	status := "online"
	if !d.Alive {
		status = "offline"
	}
	rttClass := "rtt-fast"
	if d.RTT > 50*time.Millisecond {
		rttClass = "rtt-medium"
	}
	if d.RTT > 200*time.Millisecond {
		rttClass = "rtt-slow"
	}

	buf.WriteString(fmt.Sprintf(`<tr id="%s" class="%s">
  <td>%s</td>
  <td>%s</td>
  <td class="%s">%.1fms</td>
  <td>%s</td>
  <td>%s</td>
  <td>`, d.SafeID(), status, html.EscapeString(d.IP), status, rttClass, d.RTTms(), html.EscapeString(d.Vendor), html.EscapeString(d.Hostname)))

	for _, port := range d.OpenPorts {
		buf.WriteString(fmt.Sprintf(`<span class="port">%d</span>`, port))
	}
	buf.WriteString(`</td></tr>`)
}
