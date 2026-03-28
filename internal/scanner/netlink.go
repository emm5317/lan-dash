package scanner

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/emm5317/lan-dash/internal/store"
	"github.com/vishvananda/netlink"
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

func enrich(ctx context.Context, dev store.Device, s *store.Store) {
	rtt, alive := TCPPing(ctx, dev.IP)
	dev.RTT = rtt
	dev.Alive = alive
	dev.Vendor = vendor(dev.MAC)
	if alive {
		dev.OpenPorts = ScanPorts(ctx, dev.IP)
	}
	s.Upsert(dev)
}

// Run periodically scans the ARP table and updates the store
func Run(ctx context.Context, s *store.Store) error {
	// Seed existing neighbors on startup
	existing, err := netlink.NeighList(0, 2) // AF_INET
	if err != nil {
		slog.Warn("failed to list neighbors", "err", err)
	}
	known := make(map[string]bool)
	for _, n := range existing {
		if n.State&2 != 0 && n.IP != nil { // NUD_REACHABLE
			ip := n.IP.String()
			dev := s.Upsert(store.Device{
				IP:  ip,
				MAC: n.HardwareAddr.String(),
			})
			known[ip] = true
			go enrich(ctx, dev, s)
		}
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	slog.Info("netlink scanner running")
	for {
		select {
		case <-ticker.C:
			current, err := netlink.NeighList(0, 2)
			if err != nil {
				slog.Warn("failed to list neighbors", "err", err)
				continue
			}
			currentMap := make(map[string]*netlink.Neigh)
			for _, n := range current {
				if n.IP != nil {
					currentMap[n.IP.String()] = &n
				}
			}
			// Check for new devices
			for ip, n := range currentMap {
				if !known[ip] && n.State&2 != 0 {
					dev := s.Upsert(store.Device{
						IP:    ip,
						MAC:   n.HardwareAddr.String(),
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
