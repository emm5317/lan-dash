package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/emm5317/lan-dash/internal/store"
	"time"
)

func renderTable(devices map[string]*store.Device) string {
	if len(devices) == 0 {
		return "No devices found"
	}

	table := ""
	for ip, dev := range devices {
		rttColor := getRTTColor(dev.RTT)
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render(ip),
			lipgloss.NewStyle().Width(17).Render(dev.MAC),
			lipgloss.NewStyle().Foreground(rttColor).Render(dev.RTT.String()),
		)
		table += row + "\n"
	}
	return table
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
