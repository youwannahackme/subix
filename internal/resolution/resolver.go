package resolution

import (
	"context"
	"net"
	"sync"
	"time"
)

// Resolver handles concurrent DNS resolution using custom upstream resolvers
type Resolver struct {
	threads  int
	resolver *net.Resolver
}

// NewResolver creates a new DNS resolver with custom DNS servers (Cloudflare, Google, Quad9)
func NewResolver(threads int) *Resolver {
	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
	}

	dnsServers := []string{
		"1.1.1.1:53",
		"8.8.8.8:53",
		"9.9.9.9:53",
		"1.0.0.1:53",
		"8.8.4.4:53",
	}

	var mu sync.Mutex
	dnsIndex := 0

	customResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			mu.Lock()
			server := dnsServers[dnsIndex]
			dnsIndex = (dnsIndex + 1) % len(dnsServers)
			mu.Unlock()
			return dialer.DialContext(ctx, network, server)
		},
	}

	return &Resolver{
		threads:  threads,
		resolver: customResolver,
	}
}

// Resolve performs DNS lookup for a hostname and returns IPs
func (r *Resolver) Resolve(host string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	ips, err := r.resolver.LookupHost(ctx, host)
	if err != nil {
		return nil
	}

	// Filter out IPv6 if we also have IPv4 (prefer IPv4)
	var result []string
	hasIPv4 := false
	for _, ip := range ips {
		if isIPv4(ip) {
			hasIPv4 = true
			break
		}
	}

	for _, ip := range ips {
		if hasIPv4 && !isIPv4(ip) {
			continue
		}
		result = append(result, ip)
	}

	return result
}

// ResolveBatch resolves a batch of hostnames concurrently
func (r *Resolver) ResolveBatch(hosts []string) map[string][]string {
	results := make(map[string][]string)
	var mu sync.Mutex
	semaphore := make(chan struct{}, r.threads)
	var wg sync.WaitGroup

	for _, host := range hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ips := r.Resolve(h)
			if len(ips) > 0 {
				mu.Lock()
				results[h] = ips
				mu.Unlock()
			}
		}(host)
	}

	wg.Wait()
	return results
}

func isIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() != nil
}
