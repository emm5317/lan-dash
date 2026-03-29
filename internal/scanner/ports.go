package scanner

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

var CommonPorts = []int{
	22, 80, 443, 8080, 8443,
	3000, 5000, 9000,
	32400, // Plex
	1883,  // MQTT / Home Assistant
	8123,  // Home Assistant UI
	5353,  // mDNS
}

func ScanPorts(ctx context.Context, ip string, ports []int, timeout time.Duration) []int {
	sem := make(chan struct{}, 20)
	results := make(chan int, len(ports))
	var wg sync.WaitGroup
	for _, port := range ports {
		wg.Add(1)
		sem <- struct{}{}
		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()
			tctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			conn, err := (&net.Dialer{}).DialContext(tctx, "tcp",
				fmt.Sprintf("%s:%d", ip, p))
			if err == nil {
				conn.Close()
				results <- p
			}
		}(port)
	}
	wg.Wait()
	close(results)
	var open []int
	for p := range results {
		open = append(open, p)
	}
	sort.Ints(open)
	return open
}
