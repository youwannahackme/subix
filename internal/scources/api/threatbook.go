package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// ThreatBook queries api.threatbook.cn API
type ThreatBook struct{}

// Name returns the source name
func (t *ThreatBook) Name() string {
	return "threatbook"
}

type threatbookResponse struct {
	Data struct {
		SubDomains []string `json:"sub_domains"`
	} `json:"data"`
}

// Run queries ThreatBook for subdomains
func (t *ThreatBook) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["threatbook"]
	if apiKey == "" {
		return nil, fmt.Errorf("threatbook requires API key")
	}

	url := fmt.Sprintf("https://api.threatbook.cn/v3/domain/subdomain?apikey=%s&domain=%s", apiKey, domain)
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
		return nil, fmt.Errorf("threatbook status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data threatbookResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, sub := range data.Data.SubDomains {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
