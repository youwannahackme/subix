package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// PassiveTotal queries RiskIQ PassiveTotal API
type PassiveTotal struct{}

// Name returns the source name
func (p *PassiveTotal) Name() string {
	return "passivetotal"
}

// ptResponse represents PassiveTotal response
type ptResponse struct {
	Results []struct {
		Hostname string `json:"hostname"`
	} `json:"results"`
}

// Run queries PassiveTotal for subdomains
func (p *PassiveTotal) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["passivetotal"]
	username := session.Config.ProviderConfig.APIKeys["passivetotal_user"]
	if apiKey == "" || username == "" {
		return nil, fmt.Errorf("passivetotal requires API key and username")
	}

	url := fmt.Sprintf("https://api.passivetotal.org/v2/dns/passive?query=%s", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("passivetotal status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result ptResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var subdomains []string
	seen := make(map[string]bool)

	for _, r := range result.Results {
		host := strings.ToLower(strings.TrimSpace(r.Hostname))
		if host != "" && strings.HasSuffix(host, "."+domain) && !seen[host] {
			seen[host] = true
			subdomains = append(subdomains, host)
		}
	}

	return subdomains, nil
}
