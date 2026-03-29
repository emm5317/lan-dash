package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// SweepSubnet sends UDP packets to every address in the local /24 subnet
// to populate the OS ARP table before the first arp -a read.
func SweepSubnet(ctx context.Context, workers int) {
	prefix, err := localSubnet()
	if err != nil {
		slog.Warn("sweep: could not detect subnet", "err", err)
		return
	}

	sweepCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("%s.%d", prefix, i)
		wg.Add(1)
		sem <- struct{}{}
		go func(addr string) {
			defer wg.Done()
			defer func() { <-sem }()
			// UDP dial to discard port — creates ARP entry, no privileges needed
			conn, err := (&net.Dialer{Timeout: 50 * time.Millisecond}).DialContext(sweepCtx, "udp", addr+":9")
			if err == nil {
				conn.Close()
			}
		}(ip)
	}
	wg.Wait()
	slog.Info("sweep complete", "prefix", prefix)
}

// localSubnet finds the first private IPv4 address and returns its /24 prefix.
func localSubnet() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			// Check for private ranges
			if ip[0] == 10 ||
				(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
				(ip[0] == 192 && ip[1] == 168) {
				return fmt.Sprintf("%d.%d.%d", ip[0], ip[1], ip[2]), nil
			}
		}
	}
	return "", fmt.Errorf("no private IPv4 subnet found")
}
