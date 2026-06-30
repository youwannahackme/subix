package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Chaos queries dns.projectdiscovery.io API
type Chaos struct{}

// Name returns the source name
func (c *Chaos) Name() string {
	return "chaos"
}

type chaosResponse struct {
	Subdomains []string `json:"subdomains"`
}

// Run queries Chaos for subdomains
func (c *Chaos) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["chaos"]
	if apiKey == "" {
		return nil, fmt.Errorf("chaos requires API key")
	}

	url := fmt.Sprintf("https://dns.projectdiscovery.io/dns/%s/subdomains", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chaos status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data chaosResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, sub := range data.Subdomains {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" {
			if !strings.HasSuffix(sub, "."+domain) && sub != domain {
				sub = sub + "." + domain
			}
			if strings.HasSuffix(sub, "."+domain) && !seen[sub] {
				seen[sub] = true
				result = append(result, sub)
			}
		}
	}
	return result, nil
}
