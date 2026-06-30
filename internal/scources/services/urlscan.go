package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// URLScan queries urlscan.io API
type URLScan struct{}

// Name returns the source name
func (u *URLScan) Name() string {
	return "urlscan"
}

// urlscanResponse represents urlscan.io search response
type urlscanResponse struct {
	Results []struct {
		Task struct {
			Domain string `json:"domain"`
		} `json:"task"`
		Page struct {
			Domain string `json:"domain"`
		} `json:"page"`
	} `json:"results"`
}

// Run queries URLScan for subdomains
func (u *URLScan) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://urlscan.io/api/v1/search/?q=domain:%s", domain)

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
		return nil, fmt.Errorf("urlscan returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result urlscanResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var subdomains []string
	seen := make(map[string]bool)

	for _, r := range result.Results {
		d := r.Task.Domain
		if d == "" {
			d = r.Page.Domain
		}
		d = strings.ToLower(d)
		if d != "" && strings.HasSuffix(d, "."+domain) && !seen[d] {
			seen[d] = true
			subdomains = append(subdomains, d)
		}
	}

	return subdomains, nil
}
