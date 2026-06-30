package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// ThreatCrowd queries ThreatCrowd API
type ThreatCrowd struct{}

// Name returns the source name
func (t *ThreatCrowd) Name() string {
	return "threatcrowd"
}

// threatCrowdResponse represents the API response
type threatCrowdResponse struct {
	Subdomains []string `json:"subdomains"`
}

// Run queries ThreatCrowd for subdomains
func (t *ThreatCrowd) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://www.threatcrowd.org/searchApi/v2/domain/report/?domain=%s", domain)

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
		return nil, fmt.Errorf("threatcrowd returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result threatCrowdResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var subdomains []string
	seen := make(map[string]bool)

	for _, sub := range result.Subdomains {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			subdomains = append(subdomains, sub)
		}
	}

	return subdomains, nil
}
