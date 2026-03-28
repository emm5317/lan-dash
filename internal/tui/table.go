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

	table := ""
	for _, dev := range devices {
		rttColor := getRTTColor(dev.RTT)
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(15).Render(dev.IP),
			lipgloss.NewStyle().Width(17).Render(dev.MAC),
			lipgloss.NewStyle().Foreground(rttColor).Render(fmt.Sprintf("%.2fms", dev.RTTms())),
			lipgloss.NewStyle().Width(10).Render(dev.Hostname),
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
