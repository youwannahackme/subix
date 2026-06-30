package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Robtex queries api.robtex.com passive DNS
type Robtex struct{}

// Name returns the source name
func (r *Robtex) Name() string {
	return "robtex"
}

type robtexEntry struct {
	Name string `json:"name"`
}

// Run queries Robtex for subdomains
func (r *Robtex) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://api.robtex.com/pdns/forward/%s", domain)
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
		return nil, fmt.Errorf("robtex returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 || body[0] != '[' {
		return nil, fmt.Errorf("invalid JSON response (possibly blocked or service down)")
	}

	var entries []robtexEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, entry := range entries {
		sub := strings.ToLower(strings.TrimSpace(entry.Name))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
