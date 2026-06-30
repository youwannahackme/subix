package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// AlienVault queries AlienVault OTX API
type AlienVault struct{}

// Name returns the source name
func (a *AlienVault) Name() string {
	return "alienvault"
}

// otxResponse represents OTX passive DNS response
type otxResponse struct {
	PassiveDNS []struct {
		Hostname string `json:"hostname"`
	} `json:"passive_dns"`
}

// Run queries AlienVault OTX for subdomains
func (a *AlienVault) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/passive_dns", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", types.DefaultUserAgent)
	req.Header.Set("Accept", "application/json")

	apiKey := session.Config.ProviderConfig.APIKeys["alienvault"]
	if apiKey != "" {
		req.Header.Set("X-OTX-API-KEY", apiKey)
	}

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alienvault returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result otxResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var subdomains []string
	seen := make(map[string]bool)

	for _, entry := range result.PassiveDNS {
		host := strings.ToLower(strings.TrimSpace(entry.Hostname))
		if host != "" && strings.HasSuffix(host, "."+domain) && !seen[host] {
			seen[host] = true
			subdomains = append(subdomains, host)
		}
	}

	return subdomains, nil
}
