package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// FullHunt queries fullhunt.io API
type FullHunt struct{}

// Name returns the source name
func (f *FullHunt) Name() string {
	return "fullhunt"
}

type fullhuntResponse struct {
	Hosts   []string `json:"hosts"`
	Success bool     `json:"success"`
}

// Run queries FullHunt for subdomains
func (f *FullHunt) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["fullhunt"]
	if apiKey == "" {
		return nil, fmt.Errorf("fullhunt requires API key")
	}

	url := fmt.Sprintf("https://fullhunt.io/api/v1/domain/%s/subdomains", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fullhunt status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data fullhuntResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, host := range data.Hosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" && strings.HasSuffix(host, "."+domain) && !seen[host] {
			seen[host] = true
			result = append(result, host)
		}
	}
	return result, nil
}
