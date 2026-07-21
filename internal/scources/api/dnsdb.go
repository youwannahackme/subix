package api

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// DNSDB queries api.dnsdb.info for subdomains
type DNSDB struct{}

// Name returns the source name
func (d *DNSDB) Name() string {
	return "dnsdb"
}

// Run queries DNSDB API for subdomains
func (d *DNSDB) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["dnsdb"]
	if apiKey == "" {
		return nil, fmt.Errorf("dnsdb requires API token")
	}

	url := fmt.Sprintf("https://api.dnsdb.info/lookup/rrset/name/*.%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dnsdb status %d", resp.StatusCode)
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
