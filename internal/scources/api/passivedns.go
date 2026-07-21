package api

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// PassiveDNS queries api.passivedns.cn for subdomains
type PassiveDNS struct{}

// Name returns the source name
func (p *PassiveDNS) Name() string {
	return "passivedns"
}

// Run queries PassiveDNS API for subdomains
func (p *PassiveDNS) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["passivedns"]
	if apiKey == "" {
		return nil, fmt.Errorf("passivedns requires API token")
	}

	url := fmt.Sprintf("http://api.passivedns.cn/flint/rrset/*.%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-AuthToken", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("passivedns status %d", resp.StatusCode)
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
