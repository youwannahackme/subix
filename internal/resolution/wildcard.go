package resolution

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// WildcardDetector detects and filters wildcard DNS subdomains
type WildcardDetector struct {
	wildcards map[string]*wildcardInfo
	mu        sync.RWMutex
}

type wildcardInfo struct {
	detected bool
	ips      []string
}

// NewWildcardDetector creates a new wildcard detector
func NewWildcardDetector(client interface{}, timeout time.Duration) *WildcardDetector {
	_ = client
	_ = timeout
	return &WildcardDetector{
		wildcards: make(map[string]*wildcardInfo),
	}
}

// Detect checks if a domain has wildcard DNS resolution
func (w *WildcardDetector) Detect(domain string) {
	probeCount := 3
	ipSets := make([]map[string]bool, probeCount)

	for i := 0; i < probeCount; i++ {
		randomSub := generateRandomLabel(16)
		probeHost := fmt.Sprintf("%s.%s", randomSub, domain)

		ips, err := net.LookupHost(probeHost)
		if err != nil || len(ips) == 0 {
			w.mu.Lock()
			w.wildcards[domain] = &wildcardInfo{detected: false}
			w.mu.Unlock()
			return
		}

		ipSets[i] = make(map[string]bool)
		for _, ip := range ips {
			ipSets[i][ip] = true
		}
	}

	if mapsEqual(ipSets[0], ipSets[1]) && mapsEqual(ipSets[1], ipSets[2]) {
		var ipList []string
		for ip := range ipSets[0] {
			ipList = append(ipList, ip)
		}
		w.mu.Lock()
		w.wildcards[domain] = &wildcardInfo{
			detected: true,
			ips:      ipList,
		}
		w.mu.Unlock()
	} else {
		w.mu.Lock()
		w.wildcards[domain] = &wildcardInfo{detected: false}
		w.mu.Unlock()
	}
}

// IsWildcard checks if a specific subdomain matches the wildcard pattern
func (w *WildcardDetector) IsWildcard(subdomain string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	rootDomain := extractRootDomain(subdomain, w.wildcards)
	if rootDomain == "" {
		return false
	}

	info, exists := w.wildcards[rootDomain]
	if !exists || !info.detected {
		return false
	}

	ips, err := net.LookupHost(subdomain)
	if err != nil {
		return false
	}

	if len(ips) != len(info.ips) {
		return false
	}

	ipMap := make(map[string]bool)
	for _, ip := range info.ips {
		ipMap[ip] = true
	}

	for _, ip := range ips {
		if !ipMap[ip] {
			return false
		}
	}

	return true
}

func extractRootDomain(subdomain string, wildcards map[string]*wildcardInfo) string {
	for root := range wildcards {
		suffix := "." + root
		if len(subdomain) > len(suffix) && subdomain[len(subdomain)-len(suffix):] == suffix {
			return root
		}
	}
	return ""
}

func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func generateRandomLabel(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rng.Intn(len(charset))]
	}
	return string(result)
}
