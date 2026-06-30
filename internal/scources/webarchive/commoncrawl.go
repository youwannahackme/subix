package webarchive

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// CommonCrawl queries CommonCrawl Index API
type CommonCrawl struct{}

// Name returns the source name
func (c *CommonCrawl) Name() string {
	return "commoncrawl"
}

// ccEntry represents a CommonCrawl index entry
type ccEntry struct {
	URL string `json:"url"`
}

// Run queries CommonCrawl for subdomains
func (c *CommonCrawl) Run(domain string, session *types.Session) ([]string, error) {
	// First get the latest index
	indexURL := "https://index.commoncrawl.org/collinfo.json"

	req, err := http.NewRequest("GET", indexURL, nil)
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
		return nil, fmt.Errorf("commoncrawl returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var indexes []struct {
		ID  string `json:"id"`
		API string `json:"cdx-api"`
	}
	if err := json.Unmarshal(body, &indexes); err != nil {
		return nil, err
	}

	if len(indexes) == 0 {
		return nil, fmt.Errorf("no commoncrawl indexes found")
	}

	// Use the latest index
	apiURL := indexes[0].API
	searchURL := fmt.Sprintf("%s?url=*.%s*&output=json&fl=url", apiURL, domain)

	searchReq, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	searchReq.Header.Set("User-Agent", types.DefaultUserAgent)

	searchResp, err := session.DoWithRetry(searchReq)
	if err != nil {
		return nil, err
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("commoncrawl cdx search returned status %d", searchResp.StatusCode)
	}

	searchBody, err := io.ReadAll(searchResp.Body)
	if err != nil {
		return nil, err
	}

	var subdomains []string
	seen := make(map[string]bool)

	lines := strings.Split(string(searchBody), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry ccEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		host := entry.URL
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimPrefix(host, "https://")
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" && strings.HasSuffix(host, "."+domain) && !seen[host] {
			seen[host] = true
			subdomains = append(subdomains, host)
		}
	}

	return subdomains, nil
}
