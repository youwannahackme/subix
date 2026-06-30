package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// BeVigil queries osint.bevigil.com API
type BeVigil struct{}

// Name returns the source name
func (b *BeVigil) Name() string {
	return "bevigil"
}

type bevigilResponse struct {
	Subdomains []string `json:"subdomains"`
}

// Run queries BeVigil for subdomains
func (b *BeVigil) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["bevigil"]
	if apiKey == "" {
		return nil, fmt.Errorf("bevigil requires API key")
	}

	url := fmt.Sprintf("https://osint.bevigil.com/api/%s/subdomains/", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Access-Token", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bevigil status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data bevigilResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, sub := range data.Subdomains {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
