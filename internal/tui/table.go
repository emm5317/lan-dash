package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/emm5317/lan-dash/internal/store"
)

var (
	styleHeader  = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("241"))
	styleOnline  = lipgloss.NewStyle().Foreground(lipgloss.Color("#1D9E75")).Bold(true)
	styleOffline = lipgloss.NewStyle().Foreground(lipgloss.Color("#888780"))
	styleFast    = lipgloss.NewStyle().Foreground(lipgloss.Color("#1D9E75"))
	styleMedium  = lipgloss.NewStyle().Foreground(lipgloss.Color("#BA7517"))
	styleSlow    = lipgloss.NewStyle().Foreground(lipgloss.Color("#D85A30"))
	stylePort    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7F77DD"))
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleHelp    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	styleGroup   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7F77DD"))

	colIP     = 16
	colStatus = 10
	colRTT    = 30
	colVendor = 22
	colHost   = 24
	colName   = 16
	colSpeed  = 10
)

var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func sparkline(rtts []float64, width int) string {
	if len(rtts) == 0 || width == 0 {
		return ""
	}
	// Take last `width` values
	if len(rtts) > width {
		rtts = rtts[len(rtts)-width:]
	}
	min, max := rtts[0], rtts[0]
	for _, v := range rtts {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	out := make([]rune, len(rtts))
	for i, v := range rtts {
		var idx int
		if max > min {
			idx = int((v - min) / (max - min) * float64(len(sparkChars)-1))
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		out[i] = sparkChars[idx]
	}
	return string(out)
}

func (m Model) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString(
		styleHeader.Width(colIP).Render("IP") +
			styleHeader.Width(colStatus).Render("Status") +
			styleHeader.Width(colRTT).Render("RTT") +
			styleHeader.Width(colVendor).Render("Vendor") +
			styleHeader.Width(colHost).Render("Hostname") +
			styleHeader.Width(colName).Render("Name") +
			styleHeader.Width(colSpeed).Render("Speed") +
			styleHeader.Render("Ports") + "\n",
	)

	// Sort filtered devices by (GroupOrder, IP)
	devices := m.filtered()
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

	// Build a lookup from IP to original cursor index (pre-sort)
	// We track cursor against the sorted slice
	var currentGroup store.DeviceGroup
	firstGroup := true
	for i, d := range sorted {
		// Group separator
		g := d.Group()
		if firstGroup || g != currentGroup {
			sb.WriteString("  " + styleGroup.Render(fmt.Sprintf("── %s ──────────────", string(g))) + "\n")
			currentGroup = g
			firstGroup = false
		}

		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}

		status := styleOffline.Render("○ offline")
		if d.Alive {
			status = styleOnline.Render("● online")
		}

		rttStr := "—"
		rs := styleMuted
		if d.RTT > 0 {
			ms := d.RTTms()
			spark := m.sparklines[d.IP]
			if spark != "" {
				rttStr = fmt.Sprintf("%.1fms %s", ms, spark)
			} else {
				rttStr = fmt.Sprintf("%.1fms", ms)
			}
			rs = rttStyle(d.RTT)
		}

		var ports string
		for _, p := range d.OpenPorts {
			ports += stylePort.Render(fmt.Sprintf(":%d", p)) + " "
		}
		if ports == "" {
			ports = styleMuted.Render("—")
		}

		vendor := d.Vendor
		if vendor == "" {
			vendor = styleMuted.Render("unknown")
		}
		host := d.Hostname
		if host == "" {
			host = styleMuted.Render("—")
		}

		nick := ""
		if m.nicknames != nil {
			nick = m.nicknames.Get(d.IP)
		}
		nameDisplay := nick
		if nameDisplay == "" {
			nameDisplay = styleMuted.Render("—")
		}

		speedText := "—"
		speedStyle := styleMuted
		switch d.Speed {
		case "Excellent", "Good":
			speedStyle = styleFast
			speedText = d.Speed
		case "Fair":
			speedStyle = styleMedium
			speedText = d.Speed
		case "Poor":
			speedStyle = styleSlow
			speedText = d.Speed
		}

		sb.WriteString(
			cursor +
				lipgloss.NewStyle().Width(colIP-2).Render(d.IP) +
				lipgloss.NewStyle().Width(colStatus).Render(status) +
				rs.Width(colRTT).Render(rttStr) +
				lipgloss.NewStyle().Width(colVendor).Render(truncate(vendor, colVendor)) +
				lipgloss.NewStyle().Width(colHost).Render(truncate(host, colHost)) +
				lipgloss.NewStyle().Width(colName).Render(truncate(nameDisplay, colName)) +
				speedStyle.Width(colSpeed).Render(truncate(speedText, colSpeed)) +
				ports + "\n",
		)
	}

	if len(sorted) == 0 {
		sb.WriteString(styleMuted.Render("  No devices found") + "\n")
	}

	// Edit prompt
	if m.editing {
		devs := m.filtered()
		ip := ""
		if m.cursor < len(devs) {
			ip = devs[m.cursor].IP
		}
		sb.WriteString(styleHelp.Render(fmt.Sprintf("name for %s: %s█", ip, m.editInput)) + "\n")
	} else if m.filtering {
		sb.WriteString(styleHelp.Render("/ "+m.filter+"█") + "\n")
	} else if m.filter != "" {
		sb.WriteString(styleMuted.Render("/ "+m.filter) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleHelp.Render("j/k navigate · / filter · n name · s scan · q quit"))

	return sb.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func rttStyle(rtt time.Duration) lipgloss.Style {
	switch {
	case rtt <= 50*time.Millisecond:
		return styleFast
	case rtt <= 150*time.Millisecond:
		return styleMedium
	default:
		return styleSlow
	}
}
