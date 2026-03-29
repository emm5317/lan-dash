package scanner

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/emm5317/lan-dash/internal/store"
)

// DiscoverMDNS queries mDNS services and upserts resolved hostnames into the store.
// Best-effort: returns silently if mDNS is unavailable on the network.
func DiscoverMDNS(ctx context.Context, s *store.Store, timeout time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("mdns: recovered from panic", "err", r)
		}
	}()

	services := []string{"_http._tcp", "_ssh._tcp", "_workstation._tcp"}
	entries := make(chan *mdns.ServiceEntry, 32)

	// Collect entries in background
	found := make(map[string]string) // ip → hostname
	done := make(chan struct{})
	go func() {
		defer close(done)
		for entry := range entries {
			if entry.AddrV4 != nil {
				host := strings.TrimSuffix(entry.Host, ".")
				if host != "" {
					found[entry.AddrV4.String()] = host
				}
			}
		}
	}()

	for _, svc := range services {
		params := mdns.DefaultParams(svc)
		params.Timeout = timeout
		params.Entries = entries
		if err := mdns.Query(params); err != nil {
			slog.Debug("mdns: query failed", "service", svc, "err", err)
		}
	}
	close(entries)
	<-done

	for ip, host := range found {
		s.Upsert(store.Device{IP: ip, Hostname: host})
	}
	if len(found) > 0 {
		slog.Info("mdns: discovered hostnames", "count", len(found))
	}
}
