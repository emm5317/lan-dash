package scanner

import (
	"github.com/emm5317/lan-dash/internal/store"
	"net"
	"strconv"
	"sync"
	"time"
)

func (s *Scanner) ScanPorts(ip string) {
	commonPorts := []int{22, 80, 443, 8080} // etc.
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // semaphore for 10 concurrent scans

	for _, port := range commonPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			addr := net.JoinHostPort(ip, strconv.Itoa(p))
			conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
			if err == nil {
				conn.Close()
				// Get current device and add port
				devices := s.store.All()
				for _, d := range devices {
					if d.IP == ip {
						d.OpenPorts = append(d.OpenPorts, p)
						s.store.Upsert(d)
						break
					}
				}
			}
		}(port)
	}
	wg.Wait()
}
