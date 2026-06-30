package certtransparency

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Certspotter queries Certspotter API
type Certspotter struct{}

// Name returns the source name
func (c *Certspotter) Name() string {
	return "certspotter"
}

// certspotterEntry represents a Certspotter issuance
type certspotterEntry struct {
	DNSNames []string `json:"dns_names"`
}

// Run queries Certspotter for subdomains
func (c *Certspotter) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://api.certspotter.com/v1/issuances?domain=%s&include_subdomains=true&expand=dns_names", domain)

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
		return nil, fmt.Errorf("certspotter returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 || body[0] != '[' {
		return nil, fmt.Errorf("invalid JSON response (possibly blocked or service down)")
	}

	var entries []certspotterEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}

	var subdomains []string
	seen := make(map[string]bool)

	for _, entry := range entries {
		for _, name := range entry.DNSNames {
			name = strings.TrimSpace(name)
			name = strings.TrimPrefix(name, "*.")
			if name != "" && strings.HasSuffix(name, "."+domain) && !seen[name] {
				seen[name] = true
				subdomains = append(subdomains, name)
			}
		}
	}

	return subdomains, nil
}
