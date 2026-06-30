package types

import (
	"net/http"
	"strings"
	"time"
)

// DefaultUserAgent is a standard browser User-Agent to avoid blocks/403s
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36"


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
	Config *Config
	Client *http.Client
}

// DoWithRetry executes an HTTP request and retries on temporary network errors, DNS failures, and rate limits (429/502/503/504)
func (s *Session) DoWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < 3; i++ {
		resp, err = s.Client.Do(req)
		if err == nil {
			// Check status codes that warrant a retry
			if resp.StatusCode == http.StatusOK {
				return resp, nil
			}
			if resp.StatusCode == http.StatusTooManyRequests || 
				resp.StatusCode == http.StatusBadGateway || 
				resp.StatusCode == http.StatusServiceUnavailable || 
				resp.StatusCode == http.StatusGatewayTimeout {
				resp.Body.Close()
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				continue
			}
			// For any other status code, return it so the caller can handle it
			return resp, nil
		}

		// It's a network error. Check if it's temporary, DNS, or timeout
		errStr := err.Error()
		isTemporary := strings.Contains(errStr, "timeout") || 
			strings.Contains(errStr, "deadline") || 
			strings.Contains(errStr, "lookup") || 
			strings.Contains(errStr, "connection refused") || 
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "getaddrinfow")

		if isTemporary {
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
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

// SubdomainResult represents a discovered subdomain
type SubdomainResult struct {
	Host   string   `json:"host"`
	Source string   `json:"source"`
	IPs    []string `json:"ips,omitempty"`
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
}

// NewStats creates initialized stats
func NewStats() *Stats {
	return &Stats{
		SourceCount: make(map[string]int),
		Errors:      make(map[string]int),
	}
}
