package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// SecurityTrails queries SecurityTrails API
type SecurityTrails struct{}

// Name returns the source name
func (s *SecurityTrails) Name() string {
	return "securitytrails"
}

// stResponse represents SecurityTrails API response
type stResponse struct {
	Subdomains []string `json:"subdomains"`
}

// Run queries SecurityTrails for subdomains
func (s *SecurityTrails) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["securitytrails"]
	if apiKey == "" {
		return nil, fmt.Errorf("securitytrails requires API key")
	}

	url := fmt.Sprintf("https://api.securitytrails.com/v1/domain/%s/subdomains", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("APIKEY", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("securitytrails status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result stResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var subdomains []string
	for _, sub := range result.Subdomains {
		full := strings.ToLower(sub + "." + domain)
		subdomains = append(subdomains, full)
	}

	return subdomains, nil
}
