package types

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent is a standard browser User-Agent to avoid blocks/403s
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36"

// RateLimiter interface for rate limiting HTTP requests
type RateLimiter interface {
	Wait()
	WaitContext(ctx context.Context) error
	TryWait() bool
}

// Config holds all runtime configuration
type Config struct {
	Threads         int
	Timeout         time.Duration
	MaxDepth        int
	RateLimit       int
	ResolveDNS      bool
	Recursive       bool
	Permutation     bool
	WildcardFilter  bool
	OutputFormat    string
	OutputFile      string
	AllSources      bool
	OnlyResolved    bool
	RemoveDuplicate bool
	Silent          bool
	ShowStats       bool
	ConfigPath      string
	Wordlist        string
	IncludeSources  []string
	ExcludeSources  []string
	ProviderConfig  *ProviderConfig
}

// ProviderConfig holds API keys and source enable/disable settings
type ProviderConfig struct {
	Sources map[string]map[string]bool `yaml:"sources"`
	APIKeys map[string]string          `yaml:"apikeys"`
	Censys  CensysConfig               `yaml:"censys"`
}

// CensysConfig holds Censys-specific auth
type CensysConfig struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
}

// Session holds per-source runtime state
type Session struct {
	Config      *Config
	Client      *http.Client
	Context     context.Context
	RateLimiter RateLimiter
	SourceName  string
	Stats       *Stats
}

// DoWithRetry executes an HTTP request and retries on temporary network errors, DNS failures, and rate limits (429/502/503/504)
func (s *Session) DoWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Ensure request carries the session context if none attached
	if s.Context != nil && (req.Context() == nil || req.Context() == context.Background()) {
		req = req.WithContext(s.Context)
	}

	for i := 0; i < 3; i++ {
		// Check context before each attempt
		if req.Context() != nil {
			if ctxErr := req.Context().Err(); ctxErr != nil {
				return nil, ctxErr
			}
		}

		// Apply rate limiter if configured
		if s.RateLimiter != nil {
			if limitErr := s.RateLimiter.WaitContext(req.Context()); limitErr != nil {
				return nil, limitErr
			}
		}

		if i > 0 && req.GetBody != nil {
			newBody, errBody := req.GetBody()
			if errBody == nil {
				req.Body = newBody
			}
		}

		resp, err = s.Client.Do(req)
		if err == nil {
			// Check status codes that warrant a retry
			if resp.StatusCode == http.StatusOK {
				return resp, nil
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				if s.Stats != nil && s.SourceName != "" {
					s.Stats.RecordRateLimit(s.SourceName)
				}
				resp.Body.Close()
				backoff := time.Duration(i+1) * 2 * time.Second
				if waitErr := sleepWithContext(req.Context(), backoff); waitErr != nil {
					return nil, waitErr
				}
				continue
			}

			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
				if s.Stats != nil && s.SourceName != "" {
					s.Stats.RecordBlocked(s.SourceName)
				}
				return resp, nil
			}

			if resp.StatusCode == http.StatusBadGateway ||
				resp.StatusCode == http.StatusServiceUnavailable ||
				resp.StatusCode == http.StatusGatewayTimeout {
				resp.Body.Close()
				backoff := time.Duration(i+1) * 2 * time.Second
				if waitErr := sleepWithContext(req.Context(), backoff); waitErr != nil {
					return nil, waitErr
				}
				continue
			}

			// For any other status code, return it so the caller can handle it
			return resp, nil
		}

		// It's a network error. Check if context was cancelled
		if req.Context() != nil && req.Context().Err() != nil {
			return nil, req.Context().Err()
		}

		// Check if it's temporary, DNS, or timeout
		errStr := err.Error()
		isTemporary := strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "deadline") ||
			strings.Contains(errStr, "lookup") ||
			strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "getaddrinfow")

		if isTemporary {
			backoff := time.Duration(i+1) * 2 * time.Second
			if waitErr := sleepWithContext(req.Context(), backoff); waitErr != nil {
				return nil, waitErr
			}
			continue
		}

		// Non-temporary network error, return immediately
		return nil, err
	}

	// If we exhausted all retries and still have an error, return it
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if ctx == nil {
		time.Sleep(d)
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// SubdomainResult represents a discovered subdomain
type SubdomainResult struct {
	Host   string   `json:"host"`
	Source string   `json:"source"`
	IPs    []string `json:"ips,omitempty"`
	A      []string `json:"a,omitempty"`
	AAAA   []string `json:"aaaa,omitempty"`
	CNAME  []string `json:"cname,omitempty"`
}

// Stats holds enumeration statistics
type Stats struct {
	Domains        int
	SourcesUsed    int
	TotalFound     int64
	UniqueSubs     int64
	Resolved       int64
	WildcardFilter int64
	Duration       time.Duration
	SourceCount    map[string]int
	Errors         map[string]int
	RateLimited    map[string]int
	Blocked        map[string]int
	mu             sync.Mutex
}

// NewStats creates initialized stats
func NewStats() *Stats {
	return &Stats{
		SourceCount: make(map[string]int),
		Errors:      make(map[string]int),
		RateLimited: make(map[string]int),
		Blocked:     make(map[string]int),
	}
}

// RecordRateLimit records a 429 / rate-limited event for a source
func (s *Stats) RecordRateLimit(source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.RateLimited == nil {
		s.RateLimited = make(map[string]int)
	}
	s.RateLimited[source]++
}

// RecordBlocked records a 403 / 401 blocked event for a source
func (s *Stats) RecordBlocked(source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Blocked == nil {
		s.Blocked = make(map[string]int)
	}
	s.Blocked[source]++
}
