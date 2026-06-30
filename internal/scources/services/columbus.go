package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Columbus queries columbus.elmasy.com API
type Columbus struct{}

// Name returns the source name
func (c *Columbus) Name() string {
	return "columbus"
}

// Run queries Columbus for subdomains
func (c *Columbus) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://columbus.elmasy.com/api/lookup/%s", domain)
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
		return nil, fmt.Errorf("columbus returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 || body[0] != '[' {
		return nil, fmt.Errorf("invalid JSON response (possibly blocked or service down)")
	}

	var subdomains []string
	if err := json.Unmarshal(body, &subdomains); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, sub := range subdomains {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" {
			if !strings.HasSuffix(sub, "."+domain) && sub != domain {
				sub = sub + "." + domain
			}
			if strings.HasSuffix(sub, "."+domain) && !seen[sub] {
				seen[sub] = true
				result = append(result, sub)
			}
		}
	}
	return result, nil
}
