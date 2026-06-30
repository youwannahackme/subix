package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Hunter queries api.hunter.io API
type Hunter struct{}

// Name returns the source name
func (h *Hunter) Name() string {
	return "hunter"
}

type hunterResponse struct {
	Data struct {
		Emails []struct {
			Value string `json:"value"`
		} `json:"emails"`
	} `json:"data"`
}

// Run queries Hunter for subdomains
func (h *Hunter) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["hunter"]
	if apiKey == "" {
		return nil, fmt.Errorf("hunter requires API key")
	}

	url := fmt.Sprintf("https://api.hunter.io/v2/domain-search?domain=%s&api_key=%s", domain, apiKey)
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
		return nil, fmt.Errorf("hunter status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data hunterResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, email := range data.Data.Emails {
		parts := strings.Split(email.Value, "@")
		if len(parts) == 2 {
			host := strings.ToLower(strings.TrimSpace(parts[1]))
			if host != "" && strings.HasSuffix(host, "."+domain) && !seen[host] {
				seen[host] = true
				result = append(result, host)
			}
		}
	}
	return result, nil
}
