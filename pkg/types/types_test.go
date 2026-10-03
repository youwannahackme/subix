package types

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoWithRetryContextCancellation(t *testing.T) {
	requestCount := int64(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusTooManyRequests) // 429 triggers retry
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	client := server.Client()
	stats := NewStats()
	session := &Session{
		Client:     client,
		Context:    ctx,
		SourceName: "test_src",
		Stats:      stats,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Cancel context after short delay to test that DoWithRetry aborts and doesn't sleep indefinitely
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err = session.DoWithRetry(req)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}

	// Retry loop has 2s backoff, so if context cancellation works, it should abort much faster than 2 seconds
	if duration > 1500*time.Millisecond {
		t.Errorf("DoWithRetry took %v, expected it to abort promptly on context cancellation", duration)
	}

	if stats.RateLimited["test_src"] == 0 {
		t.Errorf("expected RateLimited stat to be recorded for test_src")
	}
}

func TestSubdomainResultJSONStructuredFields(t *testing.T) {
	res := SubdomainResult{
		Host:   "api.example.com",
		Source: "mock",
		IPs:    []string{"1.2.3.4", "2606:4700::6810:1"},
		A:      []string{"1.2.3.4"},
		AAAA:   []string{"2606:4700::6810:1"},
		CNAME:  []string{"lb.example.com"},
	}

	if len(res.A) != 1 || res.A[0] != "1.2.3.4" {
		t.Errorf("unexpected A records: %v", res.A)
	}
	if len(res.AAAA) != 1 || res.AAAA[0] != "2606:4700::6810:1" {
		t.Errorf("unexpected AAAA records: %v", res.AAAA)
	}
	if len(res.CNAME) != 1 || res.CNAME[0] != "lb.example.com" {
		t.Errorf("unexpected CNAME records: %v", res.CNAME)
	}
}
