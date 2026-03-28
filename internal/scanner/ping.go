package scanner

import (
	"context"
	"net"
	"time"
)

func TCPPing(ctx context.Context, ip string) (time.Duration, bool) {
	for _, port := range []string{"80", "443", "22", "8080"} {
		tctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()
		start := time.Now()
		conn, err := (&net.Dialer{}).DialContext(tctx, "tcp", ip+":"+port)
		if err == nil {
			conn.Close()
			return time.Since(start), true
		}
	}
	return 0, false
}
