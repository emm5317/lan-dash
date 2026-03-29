package scanner

import (
	"context"
	"net"
	"time"
)

func TCPPing(ctx context.Context, ip string, timeout time.Duration) (time.Duration, bool) {
	for _, port := range []string{"80", "443", "22", "8080"} {
		tctx, cancel := context.WithTimeout(ctx, timeout)
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

func EstimateQuality(ctx context.Context, ip string, timeout time.Duration) string {
	ports := []string{"80", "443", "22", "8080"}
	var rtts []time.Duration
	for _, port := range ports {
		tctx, cancel := context.WithTimeout(ctx, timeout)
		start := time.Now()
		conn, err := (&net.Dialer{}).DialContext(tctx, "tcp", ip+":"+port)
		cancel()
		if err == nil {
			rtts = append(rtts, time.Since(start))
			conn.Close()
		}
	}
	if len(rtts) == 0 {
		return ""
	}
	var total time.Duration
	for _, r := range rtts {
		total += r
	}
	avg := total / time.Duration(len(rtts))
	switch {
	case avg < 5*time.Millisecond:
		return "Excellent"
	case avg < 20*time.Millisecond:
		return "Good"
	case avg < 100*time.Millisecond:
		return "Fair"
	default:
		return "Poor"
	}
}
