package api

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Circl queries circl.lu passive DNS API for subdomains
type Circl struct{}

// Name returns the source name
func (c *Circl) Name() string {
	return "circl"
}

// Run queries Circl API for subdomains
func (c *Circl) Run(domain string, session *types.Session) ([]string, error) {
	user := session.Config.ProviderConfig.APIKeys["circl_user"]
	pass := session.Config.ProviderConfig.APIKeys["circl_pass"]
	if user == "" || pass == "" {
		apiKey := session.Config.ProviderConfig.APIKeys["circl"]
		if apiKey != "" && strings.Contains(apiKey, ":") {
			parts := strings.SplitN(apiKey, ":", 2)
			user = parts[0]
			pass = parts[1]
		}
	}
	if user == "" || pass == "" {
		return nil, fmt.Errorf("circl requires username and password (user:pass in apikeys.circl or circl_user and circl_pass)")
	}

	url := fmt.Sprintf("https://www.circl.lu/pdns/query/%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("circl status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))
	matches := re.FindAllString(string(body), -1)

	var result []string
	seen := make(map[string]bool)
	for _, sub := range matches {
		sub = strings.ToLower(strings.TrimSpace(sub))
		sub = strings.TrimPrefix(sub, ".")
		sub = strings.TrimPrefix(sub, "*.")
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
