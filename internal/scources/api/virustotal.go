package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// VirusTotal queries VirusTotal API
type VirusTotal struct{}

// Name returns the source name
func (v *VirusTotal) Name() string {
	return "virustotal"
}


// vtSubdomainResponse is for the subdomains endpoint
type vtSubdomainResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// Run queries VirusTotal for subdomains
func (v *VirusTotal) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["virustotal"]
	if apiKey == "" {
		return nil, fmt.Errorf("virustotal requires API key")
	}

	url := fmt.Sprintf("https://www.virustotal.com/api/v3/domains/%s/subdomains?limit=40", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apikey", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("virustotal status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result vtSubdomainResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var subdomains []string
	for _, item := range result.Data {
		sub := strings.ToLower(strings.TrimSpace(item.ID))
		if sub != "" && strings.HasSuffix(sub, "."+domain) {
			subdomains = append(subdomains, sub)
		}
	}

	return subdomains, nil
}
