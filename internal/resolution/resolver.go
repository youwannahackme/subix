package resolution

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// DNSLookup defines the interface for DNS resolution, satisfied by net.Resolver or mocks
type DNSLookup interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
}

// DNSResult contains structured DNS records for a host
type DNSResult struct {
	IPs   []string
	A     []string
	AAAA  []string
	CNAME []string
}

// Resolver handles concurrent DNS resolution using custom upstream resolvers with short-lived in-memory caching
type Resolver struct {
	threads  int
	resolver DNSLookup
	cache    sync.Map // host string -> *DNSResult
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

	return NewResolverWithLookup(threads, customResolver)
}

// NewResolverWithLookup creates a Resolver with a custom or mocked DNSLookup implementation
func NewResolverWithLookup(threads int, lookup DNSLookup) *Resolver {
	if threads <= 0 {
		threads = 10
	}
	return &Resolver{
		threads:  threads,
		resolver: lookup,
	}
}

// Resolve performs DNS lookup for a hostname and returns filtered IPs (utilizing cache to prevent duplicate lookups)
func (r *Resolver) Resolve(host string) []string {
	details := r.ResolveDetails(host)
	if details == nil {
		return nil
	}
	return details.IPs
}

// ResolveDetails performs DNS lookup and returns structured records (A, AAAA, CNAME, IPs) with caching
func (r *Resolver) ResolveDetails(host string) *DNSResult {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return &DNSResult{}
	}

	// 1. Check in-memory cache first (prevents duplicate DNS resolution across wildcard filter and result pipeline)
	if val, ok := r.cache.Load(host); ok {
		if res, ok := val.(*DNSResult); ok {
			return res
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	res := &DNSResult{}

	// Perform IP lookup
	ips, err := r.resolver.LookupHost(ctx, host)

	// Perform CNAME lookup
	cname, cnameErr := r.resolver.LookupCNAME(ctx, host)
	if cnameErr == nil {
		cnameClean := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(cname)), ".")
		hostClean := strings.TrimSuffix(host, ".")
		if cnameClean != "" && cnameClean != hostClean {
			res.CNAME = []string{cnameClean}
		}
	}

	if err == nil && len(ips) > 0 {
		var aRecords []string
		var aaaaRecords []string
		var resultIPs []string
		hasIPv4 := false

		for _, ip := range ips {
			if isIPv4(ip) {
				hasIPv4 = true
				aRecords = append(aRecords, ip)
			} else {
				aaaaRecords = append(aaaaRecords, ip)
			}
		}

		// Filter out IPv6 if IPv4 is available (prefer IPv4)
		for _, ip := range ips {
			if hasIPv4 && !isIPv4(ip) {
				continue
			}
			resultIPs = append(resultIPs, ip)
		}

		res.IPs = resultIPs
		res.A = aRecords
		res.AAAA = aaaaRecords
	}

	r.cache.Store(host, res)
	return res
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
