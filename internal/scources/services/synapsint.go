package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Synapsint queries synapsint.com API
type Synapsint struct{}

// Name returns the source name
func (s *Synapsint) Name() string {
	return "synapsint"
}

// Run queries Synapsint for subdomains
func (s *Synapsint) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://synapsint.com/query.php?inputs=%s", domain)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Synapsint API can return varying JSON structures or raw text. We use a robust regex fallback to extract subdomains.
	var result []string
	seen := make(map[string]bool)

	// Try JSON first
	var data struct {
		Subdomains []string `json:"subdomains"`
	}
	if err := json.Unmarshal(body, &data); err == nil && len(data.Subdomains) > 0 {
		for _, sub := range data.Subdomains {
			sub = strings.ToLower(strings.TrimSpace(sub))
			if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
				seen[sub] = true
				result = append(result, sub)
			}
		}
		return result, nil
	}

	// Regex fallback on the body
	re := regexp.MustCompile(`(?i)[a-z0-9-]+\.` + regexp.QuoteMeta(domain))
	matches := re.FindAllString(string(body), -1)
	for _, sub := range matches {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}

	return result, nil
}
