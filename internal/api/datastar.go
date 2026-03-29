package api

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/emm5317/lan-dash/internal/nickname"
	"github.com/emm5317/lan-dash/internal/store"
)

func DatastarHandler(s *store.Store, nicks *nickname.Store) http.HandlerFunc {
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
		renderTable(&buf, s.All(), nicks)
		buf.WriteString(`</tbody>`)
		writeFragment(w, "#device-tbody", "morph", buf.String())
		flusher.Flush()

		events, unsub := s.Subscribe()
		defer unsub()

		for {
			select {
			case ev := <-events:
				buf.Reset()
				nick := ""
				if nicks != nil {
					nick = nicks.Get(ev.Device.IP)
				}
				renderRow(&buf, ev.Device, nick)
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

func renderTable(buf *bytes.Buffer, devices []store.Device, nicks *nickname.Store) {
	// Sort by group order, then by IP
	sorted := make([]store.Device, len(devices))
	copy(sorted, devices)
	sort.Slice(sorted, func(i, j int) bool {
		gi := store.GroupOrder[sorted[i].Group()]
		gj := store.GroupOrder[sorted[j].Group()]
		if gi != gj {
			return gi < gj
		}
		return sorted[i].IP < sorted[j].IP
	})

	var currentGroup store.DeviceGroup
	first := true
	for _, d := range sorted {
		g := d.Group()
		if first || g != currentGroup {
			// Insert group header row
			buf.WriteString(fmt.Sprintf(
				`<tr class="group-header"><td colspan="8">%s</td></tr>`,
				html.EscapeString(string(g)),
			))
			currentGroup = g
			first = false
		}
		nick := ""
		if nicks != nil {
			nick = nicks.Get(d.IP)
		}
		renderRow(buf, d, nick)
	}
}

func renderRow(buf *bytes.Buffer, d store.Device, nickname string) {
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

	speedClass := ""
	speedVal := d.Speed
	switch d.Speed {
	case "Excellent", "Good":
		speedClass = "rtt-fast"
	case "Fair":
		speedClass = "rtt-medium"
	case "Poor":
		speedClass = "rtt-slow"
	default:
		speedVal = "—"
	}

	buf.WriteString(fmt.Sprintf(`<tr id="%s" class="%s">
  <td>%s</td>
  <td>%s</td>
  <td class="%s">%.1fms</td>
  <td>%s</td>
  <td>%s</td>
  <td>%s</td>
  <td class="%s">%s</td>
  <td>`, d.SafeID(), status,
		html.EscapeString(d.IP),
		status,
		rttClass, d.RTTms(),
		html.EscapeString(d.Vendor),
		html.EscapeString(d.Hostname),
		html.EscapeString(nickname),
		speedClass, html.EscapeString(speedVal),
	))

	for _, port := range d.OpenPorts {
		buf.WriteString(fmt.Sprintf(`<span class="port">%d</span>`, port))
	}
	buf.WriteString(`</td></tr>`)
}
