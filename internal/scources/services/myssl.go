package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// MySSL queries myssl.com subdomain discovery API
type MySSL struct{}

// Name returns the source name
func (m *MySSL) Name() string {
	return "myssl"
}

type mySSLResponse struct {
	Code int `json:"code"`
	Data []struct {
		Subdomain string `json:"subdomain"`
	} `json:"data"`
}

// Run queries MySSL for subdomains
func (m *MySSL) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://myssl.com/api/v1/discover_sub_domain?domain=%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("myssl status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data mySSLResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, entry := range data.Data {
		sub := strings.ToLower(strings.TrimSpace(entry.Subdomain))
		sub = strings.TrimPrefix(sub, ".")
		sub = strings.TrimPrefix(sub, "*.")
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
