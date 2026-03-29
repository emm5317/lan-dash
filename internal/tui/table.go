package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
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

	colIP     = 16
	colStatus = 10
	colRTT    = 9
	colVendor = 22
	colHost   = 24
)

func (m Model) View() string {
	var sb strings.Builder

	// Header
	sb.WriteString(
		styleHeader.Width(colIP).Render("IP") +
			styleHeader.Width(colStatus).Render("Status") +
			styleHeader.Width(colRTT).Render("RTT") +
			styleHeader.Width(colVendor).Render("Vendor") +
			styleHeader.Width(colHost).Render("Hostname") +
			styleHeader.Render("Ports") + "\n",
	)

	devices := m.filtered()
	for i, d := range devices {
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
			rttStr = fmt.Sprintf("%.1fms", ms)
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

		sb.WriteString(
			cursor +
				lipgloss.NewStyle().Width(colIP-2).Render(d.IP) +
				lipgloss.NewStyle().Width(colStatus).Render(status) +
				rs.Width(colRTT).Render(rttStr) +
				lipgloss.NewStyle().Width(colVendor).Render(truncate(vendor, colVendor)) +
				lipgloss.NewStyle().Width(colHost).Render(truncate(host, colHost)) +
				ports + "\n",
		)
	}

	if len(devices) == 0 {
		sb.WriteString(styleMuted.Render("  No devices found") + "\n")
	}

	// Filter prompt
	if m.filtering {
		sb.WriteString(styleHelp.Render("/ "+m.filter+"��") + "\n")
	} else if m.filter != "" {
		sb.WriteString(styleMuted.Render("/ "+m.filter) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleHelp.Render("j/k navigate · / filter · s scan · q quit"))

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
