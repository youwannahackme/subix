package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Shodan queries Shodan DNS API
type Shodan struct{}

// Name returns the source name
func (s *Shodan) Name() string {
	return "shodan"
}

// shodanDNSResponse represents Shodan DNS API response
type shodanDNSResponse map[string][]string

// Run queries Shodan for subdomains
func (s *Shodan) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["shodan"]
	if apiKey == "" {
		return nil, fmt.Errorf("shodan requires API key")
	}

	url := fmt.Sprintf("https://api.shodan.io/dns/domain/%s?key=%s", domain, apiKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("shodan status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result shodanDNSResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var subdomains []string
	for sub := range result {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" && strings.HasSuffix(sub, "."+domain) {
			subdomains = append(subdomains, sub)
		}
	}

	return subdomains, nil
}
