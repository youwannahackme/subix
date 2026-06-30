package webarchive

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Wayback queries Wayback Machine CDX API
type Wayback struct{}

// Name returns the source name
func (w *Wayback) Name() string {
	return "wayback"
}

// Run queries Wayback Machine for subdomains
func (w *Wayback) Run(domain string, session *types.Session) ([]string, error) {
	// Optimization: Using matchType=domain is the standard and most reliable way
	// to query the CDX API for a target domain and all its subdomains.
	url := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%s&matchType=domain&output=json&fl=original&collapse=urlkey", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wayback returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// First element is the header ["original"]
	var raw [][]string
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var subdomains []string
	seen := make(map[string]bool)
	domain = strings.ToLower(strings.TrimSpace(domain))

	for i, entry := range raw {
		if i == 0 {
			continue // skip header
		}
		if len(entry) < 1 {
			continue
		}

		// Convert to lowercase early to safely handle case variations in URLs
		host := strings.ToLower(strings.TrimSpace(entry[0]))
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimPrefix(host, "https://")

		// Fix 1: Strip paths and query strings safely
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		if idx := strings.Index(host, "?"); idx != -1 {
			host = host[:idx]
		}

		// Fix 2: Strip out explicit port numbers (e.g., target.com:8443 -> target.com)
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}

		host = strings.TrimSpace(host)
		host = strings.TrimPrefix(host, "www.")

		if host == "" {
			continue
		}

		// Fix 3: Allow exact matches on the root domain as well as nested subdomains
		if host == domain || strings.HasSuffix(host, "."+domain) {
			if !seen[host] {
				seen[host] = true
				subdomains = append(subdomains, host)
			}
		}
	}

	return subdomains, nil
}
