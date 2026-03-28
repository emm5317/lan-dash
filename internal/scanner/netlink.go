package scanner

import (
	"log"
	"time"

	"github.com/emm5317/lan-dash/internal/store"
	"github.com/vishvananda/netlink"
)

type Scanner struct {
	store *store.Store
}

func NewScanner(s *store.Store) *Scanner {
	return &Scanner{store: s}
}

func (s *Scanner) StartARP() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		neighs, err := netlink.NeighList(0, 0)
		if err != nil {
			log.Printf("Failed to list neighbors: %v", err)
			continue
		}
		for _, neigh := range neighs {
			// Process neighbor
			ip := neigh.IP.String()
			mac := neigh.HardwareAddr.String()
			s.store.UpdateDeviceFromARP(ip, mac)
		}
	}
}
