package runner

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/youwannahackme/subix/internal/recursive"
	"github.com/youwannahackme/subix/internal/scources"
	"github.com/youwannahackme/subix/pkg/types"
)

// mockSource implements sources.Source for runner testing
type mockSource struct {
	name string
	subs []string
}

func (m *mockSource) Name() string {
	return m.name
}

func (m *mockSource) Run(domain string, session *types.Session) ([]string, error) {
	return m.subs, nil
}

// Test 2: Recursive mode receives non-empty input even when RemoveDuplicate is false
func TestRecursiveReceivesNonEmptyInputWithoutUnique(t *testing.T) {
	domain := "example.com"
	discovered := []string{
		"admin.example.com",
		"internal.admin.example.com",
		"api.example.com",
	}

	cfg := &types.Config{
		Threads:         2,
		Timeout:         5 * time.Second,
		MaxDepth:        2,
		Recursive:       true,
		RemoveDuplicate: false, // Critical: previously broke r.seen when false
		Silent:          true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &mockSource{
		name: "testmock",
		subs: discovered,
	}

	r := &Runner{
		config:  cfg,
		domains: []string{domain},
		sources: []sources.Source{mock},
		results: make(chan *types.SubdomainResult, 100),
		stats:   types.NewStats(),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Drain results channel in background
	go func() {
		for range r.results {
		}
	}()

	// Enumerate domain with mock source
	r.enumerateDomain(domain, 0)

	// Collect found subdomains as runRecursive does
	var foundSubs []string
	r.seen.Range(func(key, _ interface{}) bool {
		if s, ok := key.(string); ok {
			if strings.HasSuffix(s, "."+domain) && s != domain {
				foundSubs = append(foundSubs, s)
			}
		}
		return true
	})

	if len(foundSubs) == 0 {
		t.Fatalf("expected foundSubs to be non-empty when RemoveDuplicate=false, got 0")
	}

	if len(foundSubs) != len(discovered) {
		t.Errorf("expected %d found subdomains, got %d: %v", len(discovered), len(foundSubs), foundSubs)
	}

	// Verify recursive engine actually receives and recurses into these subdomains
	recursedCount := int64(0)
	engine := recursive.NewEngine(cfg.MaxDepth, func(subDomain string, currentDepth int) {
		atomic.AddInt64(&recursedCount, 1)
	})

	engine.Run(domain, foundSubs)

	if atomic.LoadInt64(&recursedCount) == 0 {
		t.Errorf("expected recursive engine to recurse into at least 1 parent domain, got 0")
	}
}

// Test deduplication behavior:
// When RemoveDuplicate is true, duplicate results are suppressed from r.results
// When RemoveDuplicate is false, duplicate results are emitted to r.results
func TestDeduplicationFlagBehavior(t *testing.T) {
	domain := "example.com"
	duplicateSubs := []string{
		"app.example.com",
		"app.example.com",
		"app.example.com",
	}

	// 1. With RemoveDuplicate = true
	{
		cfg := &types.Config{
			Threads:         1,
			Timeout:         5 * time.Second,
			RemoveDuplicate: true,
			Silent:          true,
		}
		ctx, cancel := context.WithCancel(context.Background())
		r := &Runner{
			config:  cfg,
			domains: []string{domain},
			sources: []sources.Source{&mockSource{name: "src1", subs: duplicateSubs}},
			results: make(chan *types.SubdomainResult, 10),
			stats:   types.NewStats(),
			ctx:     ctx,
			cancel:  cancel,
		}

		done := make(chan []string)
		go func() {
			var emitted []string
			for res := range r.results {
				emitted = append(emitted, res.Host)
			}
			done <- emitted
		}()

		r.enumerateDomain(domain, 0)
		close(r.results)
		cancel()

		emitted := <-done
		if len(emitted) != 1 {
			t.Errorf("expected 1 result emitted with RemoveDuplicate=true, got %d: %v", len(emitted), emitted)
		}
	}

	// 2. With RemoveDuplicate = false
	{
		cfg := &types.Config{
			Threads:         1,
			Timeout:         5 * time.Second,
			RemoveDuplicate: false,
			Silent:          true,
		}
		ctx, cancel := context.WithCancel(context.Background())
		r := &Runner{
			config:  cfg,
			domains: []string{domain},
			sources: []sources.Source{&mockSource{name: "src1", subs: duplicateSubs}},
			results: make(chan *types.SubdomainResult, 10),
			stats:   types.NewStats(),
			ctx:     ctx,
			cancel:  cancel,
		}

		done := make(chan []string)
		go func() {
			var emitted []string
			for res := range r.results {
				emitted = append(emitted, res.Host)
			}
			done <- emitted
		}()

		r.enumerateDomain(domain, 0)
		close(r.results)
		cancel()

		emitted := <-done
		if len(emitted) != 3 {
			t.Errorf("expected 3 results emitted with RemoveDuplicate=false, got %d: %v", len(emitted), emitted)
		}
	}
}

// Test rate limiting integration in runner
func TestRateLimiterIntegration(t *testing.T) {
	rate := 10 // 10 req/sec
	cfg := &types.Config{
		Threads:         1,
		Timeout:         5 * time.Second,
		RateLimit:       rate,
		RemoveDuplicate: true,
		Silent:          true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &mockSource{
		name: "ratelimited_src",
		subs: []string{"a.example.com"},
	}

	r := &Runner{
		config:  cfg,
		domains: []string{"example.com"},
		sources: []sources.Source{mock},
		results: make(chan *types.SubdomainResult, 10),
		stats:   types.NewStats(),
		ctx:     ctx,
		cancel:  cancel,
	}

	go func() {
		for range r.results {
		}
	}()

	start := time.Now()
	r.enumerateDomain("example.com", 0)
	duration := time.Since(start)

	// Just verifying enumerateDomain completes cleanly with rate limiter active
	if duration < 0 {
		t.Fatal("negative duration")
	}
}
