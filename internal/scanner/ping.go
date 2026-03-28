package scanner

import (
	"github.com/emm5317/lan-dash/internal/store"
	"net"
	"time"
)

func (s *Scanner) PingDevice(ip string) time.Duration {
	// Try TCP connect to port 80 first, then 443
	ports := []string{"80", "443"}
	for _, port := range ports {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), 1*time.Second)
		if err == nil {
			conn.Close()
			rtt := time.Since(start)
			s.store.UpdateDeviceRTT(ip, rtt)
			return rtt
		}
	}
	return 0
}
