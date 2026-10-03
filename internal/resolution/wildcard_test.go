package resolution

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// mockLookup implements DNSLookup for testing
type mockLookup struct {
	mu            sync.Mutex
	callCounts    map[string]int
	patternCounts map[string]int
	responses     map[string][][]string
	cnameAnswers  map[string]string
}

func newMockLookup() *mockLookup {
	return &mockLookup{
		callCounts:    make(map[string]int),
		patternCounts: make(map[string]int),
		responses:     make(map[string][][]string),
		cnameAnswers:  make(map[string]string),
	}
}

func (m *mockLookup) LookupHost(ctx context.Context, host string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	host = strings.ToLower(strings.TrimSpace(host))
	m.callCounts[host]++

	// 1. Check exact host match first
	if resList, ok := m.responses[host]; ok && len(resList) > 0 {
		idx := (m.callCounts[host] - 1) % len(resList)
		return resList[idx], nil
	}

	// 2. Check suffix pattern
	for pattern, resList := range m.responses {
		if strings.HasPrefix(pattern, ".") && strings.HasSuffix(host, pattern) {
			m.patternCounts[pattern]++
			idx := (m.patternCounts[pattern] - 1) % len(resList)
			return resList[idx], nil
		}
	}

	return nil, nil
}

func (m *mockLookup) LookupCNAME(ctx context.Context, host string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	host = strings.ToLower(strings.TrimSpace(host))
	for pattern, cname := range m.cnameAnswers {
		if strings.HasSuffix(host, pattern) || host == pattern {
			return cname, nil
		}
	}
	return host + ".", nil
}

// Test 3: extractRootDomain with two overlapping registered roots -> deterministic longest-match across 100 iterations
func TestExtractRootDomainLongestMatchDeterministic(t *testing.T) {
	detector := NewWildcardDetector(NewResolver(1))

	// Register overlapping roots
	detector.RegisterWildcard("example.com", true, []string{"1.1.1.1"}, nil)
	detector.RegisterWildcard("dev.example.com", true, []string{"2.2.2.2"}, nil)
	detector.RegisterWildcard("api.dev.example.com", true, []string{"3.3.3.3"}, nil)

	// Run 100 iterations to verify Go randomized map iteration does NOT cause non-determinism
	for i := 0; i < 100; i++ {
		// api.dev.example.com has 3 matching suffix roots: example.com, dev.example.com, and itself
		root := detector.ExtractRootDomain("sub.api.dev.example.com")
		if root != "api.dev.example.com" {
			t.Fatalf("iteration %d: expected longest match 'api.dev.example.com', got '%s'", i, root)
		}

		// sub.dev.example.com matches both dev.example.com and example.com
		rootDev := detector.ExtractRootDomain("sub.dev.example.com")
		if rootDev != "dev.example.com" {
			t.Fatalf("iteration %d: expected longest match 'dev.example.com', got '%s'", i, rootDev)
		}

		// foo.example.com only matches example.com
		rootEx := detector.ExtractRootDomain("foo.example.com")
		if rootEx != "example.com" {
			t.Fatalf("iteration %d: expected 'example.com', got '%s'", i, rootEx)
		}
	}
}

// Test 4: IsWildcard against a mocked resolver returning varying IP subsets across calls -> still correctly flags as wildcard
func TestIsWildcardVaryingIPSubsets(t *testing.T) {
	mock := newMockLookup()

	// Probes for .example.com return varying IP subsets (rotating CDN edge IPs)
	mock.responses[".example.com"] = [][]string{
		{"104.21.1.1", "104.21.1.2"},
		{"104.21.1.2", "104.21.1.3"},
		{"104.21.1.3", "104.21.1.4"},
	}

	res := NewResolverWithLookup(1, mock)
	detector := NewWildcardDetector(res)

	// Detect wildcards on example.com
	detector.Detect("example.com")

	// Verify detection succeeded
	detector.mu.RLock()
	info := detector.wildcards["example.com"]
	detector.mu.RUnlock()

	if info == nil || !info.detected {
		t.Fatal("expected wildcard to be detected on example.com")
	}

	// Verify union of IPs was captured
	expectedIPs := []string{"104.21.1.1", "104.21.1.2", "104.21.1.3", "104.21.1.4"}
	for _, ip := range expectedIPs {
		if !info.ips[ip] {
			t.Errorf("expected IP %s in wildcard union set", ip)
		}
	}

	// Candidate subdomain that returns a rotating subset different in length and content
	// e.g. ["104.21.1.2", "104.21.1.99"] — overlaps on 104.21.1.2
	mock.responses["wildcard-sub.example.com"] = [][]string{
		{"104.21.1.2", "104.21.1.99"},
	}

	if !detector.IsWildcard("wildcard-sub.example.com") {
		t.Error("expected candidate with overlapping IP subset to be flagged as wildcard")
	}

	// Another candidate that only returns 1 IP: 104.21.1.4
	mock.responses["single-ip.example.com"] = [][]string{
		{"104.21.1.4"},
	}

	if !detector.IsWildcard("single-ip.example.com") {
		t.Error("expected single-ip candidate matching wildcard IP to be flagged as wildcard")
	}

	// Legitimate subdomain that returns unrelated IP
	mock.responses["legit.example.com"] = [][]string{
		{"192.168.1.50"},
	}

	if detector.IsWildcard("legit.example.com") {
		t.Error("expected legitimate subdomain with non-overlapping IP NOT to be flagged as wildcard")
	}
}

// Test CNAME matching in IsWildcard
func TestIsWildcardCNAMEMatch(t *testing.T) {
	mock := newMockLookup()
	mock.responses[".cdn-wildcard.com"] = [][]string{
		{"198.51.100.1"},
	}
	mock.cnameAnswers[".cdn-wildcard.com"] = "traffic.cdnprovider.net."

	res := NewResolverWithLookup(1, mock)
	detector := NewWildcardDetector(res)
	detector.Detect("cdn-wildcard.com")

	// Candidate has different IP but same CNAME target
	mock.responses["test.cdn-wildcard.com"] = [][]string{
		{"203.0.113.5"},
	}
	mock.cnameAnswers["test.cdn-wildcard.com"] = "traffic.cdnprovider.net."

	if !detector.IsWildcard("test.cdn-wildcard.com") {
		t.Error("expected candidate with matching CNAME to be flagged as wildcard")
	}
}

// Test transient error retry during Detect
func TestWildcardDetectTransientRetry(t *testing.T) {
	mock := newMockLookup()
	// Sequence of calls: attempt 1 succeeds, attempt 2 (first try fails with empty, retry succeeds), attempt 3 succeeds
	probeResponses := [][]string{
		{"10.0.0.1"},  // probe 1
		{},            // probe 2, attempt 1: transient failure
		{"10.0.0.2"},  // probe 2, attempt 2: retry succeeds!
		{"10.0.0.1"},  // probe 3
	}
	mock.responses[".retry-domain.com"] = probeResponses

	res := NewResolverWithLookup(1, mock)
	detector := NewWildcardDetector(res)
	detector.Detect("retry-domain.com")

	detector.mu.RLock()
	info := detector.wildcards["retry-domain.com"]
	detector.mu.RUnlock()

	if info == nil || !info.detected {
		t.Fatal("expected wildcard to be detected after successful retry of transient failure")
	}
}
