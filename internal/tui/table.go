package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/emm5317/lan-dash/internal/store"
	"time"
)

func renderTable(devices []store.Device) string {
	if len(devices) == 0 {
		return "No devices found"
	}

	// Header
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Width(15).Render("IP"),
		lipgloss.NewStyle().Bold(true).Width(8).Render("Status"),
		lipgloss.NewStyle().Bold(true).Width(8).Render("RTT"),
		lipgloss.NewStyle().Bold(true).Width(12).Render("Vendor"),
		lipgloss.NewStyle().Bold(true).Width(15).Render("Hostname"),
		lipgloss.NewStyle().Bold(true).Width(20).Render("Open Ports"),
	) + "\n"

	table := header
	for _, dev := range devices {
		status := "offline"
		if dev.Alive {
			status = "online"
		}

		rttColor := getRTTColor(dev.RTT)
		rttStr := fmt.Sprintf("%.1fms", dev.RTTms())

		ports := ""
		for i, port := range dev.OpenPorts {
			if i > 0 {
				ports += " "
			}
			ports += fmt.Sprintf("%d", port)
		}
		if ports == "" {
			ports = "-"
		}

		row := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render(dev.IP),
			lipgloss.NewStyle().Width(8).Render(status),
			lipgloss.NewStyle().Foreground(rttColor).Width(8).Render(rttStr),
			lipgloss.NewStyle().Width(12).Render(truncate(dev.Vendor, 12)),
			lipgloss.NewStyle().Width(15).Render(truncate(dev.Hostname, 15)),
			lipgloss.NewStyle().Width(20).Render(truncate(ports, 20)),
		)
		table += row + "\n"
	}
	return table
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func getRTTColor(rtt time.Duration) lipgloss.Color {
	if rtt < 10*time.Millisecond {
		return lipgloss.Color("2") // green
	} else if rtt < 100*time.Millisecond {
		return lipgloss.Color("3") // yellow
	} else {
		return lipgloss.Color("1") // red
	}
}
