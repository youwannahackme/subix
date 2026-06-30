package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Netlas queries api.netlas.io API
type Netlas struct{}

// Name returns the source name
func (n *Netlas) Name() string {
	return "netlas"
}

type netlasResponse struct {
	Items []struct {
		Data struct {
			Domain string `json:"domain"`
		} `json:"data"`
	} `json:"items"`
}

// Run queries Netlas for subdomains
func (n *Netlas) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["netlas"]
	if apiKey == "" {
		return nil, fmt.Errorf("netlas requires API key")
	}

	url := fmt.Sprintf("https://api.netlas.io/responses?q=domain:*.%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("netlas status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data netlasResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, item := range data.Items {
		sub := strings.ToLower(strings.TrimSpace(item.Data.Domain))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
