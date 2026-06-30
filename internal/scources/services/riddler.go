package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Riddler queries riddler.io API
type Riddler struct{}

// Name returns the source name
func (r *Riddler) Name() string {
	return "riddler"
}

type riddlerEntry struct {
	Subdomain string `json:"subdomain"`
}

// Run queries Riddler for subdomains
func (r *Riddler) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://riddler.io/api/search?q=pki.domain:%s", domain)
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
		return nil, fmt.Errorf("riddler returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var entries []riddlerEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, entry := range entries {
		sub := strings.ToLower(strings.TrimSpace(entry.Subdomain))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
