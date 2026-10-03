package resolution

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// WildcardDetector detects and filters wildcard DNS subdomains
type WildcardDetector struct {
	resolver  *Resolver
	wildcards map[string]*wildcardInfo
	mu        sync.RWMutex
}

type wildcardInfo struct {
	detected bool
	ips      map[string]bool
	cnames   map[string]bool
}

// NewWildcardDetector creates a new wildcard detector using the shared Resolver
func NewWildcardDetector(resolver *Resolver) *WildcardDetector {
	if resolver == nil {
		resolver = NewResolver(10)
	}
	return &WildcardDetector{
		resolver:  resolver,
		wildcards: make(map[string]*wildcardInfo),
	}
}

// Detect checks if a domain has wildcard DNS resolution with retry on transient errors
func (w *WildcardDetector) Detect(domain string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	probeCount := 3
	wildcardIPs := make(map[string]bool)
	wildcardCNAMEs := make(map[string]bool)
	probesSucceeded := 0
	inconclusive := false

	for i := 0; i < probeCount; i++ {
		var details *DNSResult
		var probeErr error

		// Retry probe up to 2 times (3 attempts total) with backoff on failure
		for attempt := 0; attempt < 3; attempt++ {
			randomSub := generateRandomLabel(16)
			probeHost := fmt.Sprintf("%s.%s", randomSub, domain)

			details = w.resolver.ResolveDetails(probeHost)
			if details != nil && (len(details.IPs) > 0 || len(details.CNAME) > 0) {
				probeErr = nil
				break
			}
			probeErr = fmt.Errorf("no records returned")
			if attempt < 2 {
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			}
		}

		if probeErr != nil {
			// If probe 0 had no IPs, this domain likely does NOT have wildcard DNS.
			// But if a previous probe DID succeed and this one failed after retries,
			// or if probes conflict, it is inconclusive.
			if probesSucceeded > 0 {
				inconclusive = true
			}
			continue
		}

		probesSucceeded++
		for _, ip := range details.IPs {
			wildcardIPs[ip] = true
		}
		for _, cn := range details.CNAME {
			wildcardCNAMEs[cn] = true
		}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if inconclusive {
		fmt.Fprintf(os.Stderr, "  [%s] \033[33m! Wildcard detection inconclusive for domain %s, filtering disabled\033[0m\n", domain, domain)
		w.wildcards[domain] = &wildcardInfo{detected: false}
		return
	}

	// If all probes succeeded and returned IPs/CNAMEs, it is confirmed wildcard
	if probesSucceeded == probeCount && (len(wildcardIPs) > 0 || len(wildcardCNAMEs) > 0) {
		w.wildcards[domain] = &wildcardInfo{
			detected: true,
			ips:      wildcardIPs,
			cnames:   wildcardCNAMEs,
		}
	} else {
		w.wildcards[domain] = &wildcardInfo{detected: false}
	}
}

// IsWildcard checks if a specific subdomain matches the wildcard pattern using intersection matching
func (w *WildcardDetector) IsWildcard(subdomain string) bool {
	subdomain = strings.ToLower(strings.TrimSpace(subdomain))

	rootDomain := w.extractRootDomain(subdomain)
	if rootDomain == "" {
		return false
	}

	w.mu.RLock()
	info, exists := w.wildcards[rootDomain]
	w.mu.RUnlock()

	if !exists || !info.detected {
		return false
	}

	// Resolve the candidate subdomain using the shared resolver (populates / hits cache)
	details := w.resolver.ResolveDetails(subdomain)
	if details == nil {
		return false
	}

	// 1. Check CNAME matching: if CNAME overlaps with wildcard CNAME
	if len(info.cnames) > 0 && len(details.CNAME) > 0 {
		for _, cn := range details.CNAME {
			if info.cnames[cn] {
				return true
			}
		}
	}

	// 2. Check IP intersection: if candidate shares ANY IP with the wildcard IP set
	if len(info.ips) > 0 && len(details.IPs) > 0 {
		for _, ip := range details.IPs {
			if info.ips[ip] {
				return true
			}
		}
	}

	return false
}

// extractRootDomain deterministically finds the longest matching registered root domain
func (w *WildcardDetector) extractRootDomain(subdomain string) string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var bestRoot string
	for root := range w.wildcards {
		if subdomain == root {
			if len(root) > len(bestRoot) || (len(root) == len(bestRoot) && root > bestRoot) {
				bestRoot = root
			}
			continue
		}
		suffix := "." + root
		if strings.HasSuffix(subdomain, suffix) {
			if len(root) > len(bestRoot) || (len(root) == len(bestRoot) && root > bestRoot) {
				bestRoot = root
			}
		}
	}
	return bestRoot
}

// ExtractRootDomain exported for testing
func (w *WildcardDetector) ExtractRootDomain(subdomain string) string {
	return w.extractRootDomain(subdomain)
}

// RegisterWildcard registers a wildcard entry directly for testing
func (w *WildcardDetector) RegisterWildcard(domain string, detected bool, ips []string, cnames []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ipMap := make(map[string]bool)
	for _, ip := range ips {
		ipMap[ip] = true
	}
	cnameMap := make(map[string]bool)
	for _, cn := range cnames {
		cnameMap[cn] = true
	}

	w.wildcards[domain] = &wildcardInfo{
		detected: detected,
		ips:      ipMap,
		cnames:   cnameMap,
	}
}

func generateRandomLabel(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
