package scanner

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/emm5317/lan-dash/internal/store"
)

//go:embed oui.json
var ouiData []byte

var ouiMap map[string]string // loaded once in init()

func init() {
	json.Unmarshal(ouiData, &ouiMap)
}

func vendor(mac string) string {
	if len(mac) < 8 {
		return ""
	}
	key := strings.ToUpper(strings.ReplaceAll(mac[:8], ":", ""))
	return ouiMap[key]
}

type arpEntry struct {
	IP  string
	MAC string
}

// parseARP runs "arp -a" and parses the output into IP/MAC pairs.
// Only dynamic entries are returned (equivalent to reachable neighbors).
func parseARP() ([]arpEntry, error) {
	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		return nil, err
	}

	var entries []arpEntry
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		// Windows format: "10.0.0.1  40-0f-c1-46-45-31  dynamic"
		if len(fields) < 3 {
			continue
		}
		if !strings.EqualFold(fields[2], "dynamic") {
			continue
		}
		ip := fields[0]
		// Validate it looks like an IP
		if net.ParseIP(ip) == nil {
			continue
		}
		// Convert Windows MAC format (aa-bb-cc-dd-ee-ff) to colon-separated
		mac := strings.ReplaceAll(fields[1], "-", ":")
		entries = append(entries, arpEntry{IP: ip, MAC: mac})
	}
	return entries, nil
}

func enrich(ctx context.Context, dev store.Device, s *store.Store) {
	rtt, alive := TCPPing(ctx, dev.IP)
	dev.RTT = rtt
	dev.Alive = alive
	dev.Vendor = vendor(dev.MAC)
	if alive {
		dev.OpenPorts = ScanPorts(ctx, dev.IP)
		if names, err := net.LookupAddr(dev.IP); err == nil && len(names) > 0 {
			dev.Hostname = strings.TrimSuffix(names[0], ".")
		}
	}
	s.Upsert(dev)
}

// TriggerScan initiates enrichment for all known devices
func TriggerScan(ctx context.Context, s *store.Store) {
	for _, dev := range s.All() {
		go enrich(ctx, dev, s)
	}
}

// Run periodically scans the ARP table and updates the store
func Run(ctx context.Context, s *store.Store) error {
	// Seed existing neighbors on startup
	existing, err := parseARP()
	if err != nil {
		slog.Warn("failed to list neighbors", "err", err)
	}
	known := make(map[string]bool)
	for _, n := range existing {
		dev := s.Upsert(store.Device{
			IP:  n.IP,
			MAC: n.MAC,
		})
		known[n.IP] = true
		go enrich(ctx, dev, s)
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	slog.Info("arp scanner running")
	for {
		select {
		case <-ticker.C:
			current, err := parseARP()
			if err != nil {
				slog.Warn("failed to list neighbors", "err", err)
				continue
			}
			currentMap := make(map[string]arpEntry)
			for _, n := range current {
				currentMap[n.IP] = n
			}
			// Check for new devices
			for ip, n := range currentMap {
				if !known[ip] {
					dev := s.Upsert(store.Device{
						IP:    ip,
						MAC:   n.MAC,
						Alive: true,
					})
					slog.Info("device joined", "ip", dev.IP, "mac", dev.MAC)
					known[ip] = true
					go enrich(ctx, dev, s)
				}
			}
			// Check for offline devices
			for ip := range known {
				if _, exists := currentMap[ip]; !exists {
					slog.Info("device left", "ip", ip)
					s.SetAlive(ip, false)
					delete(known, ip)
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
