package searchengine

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Bing queries Bing search engine for subdomains
type Bing struct{}

// Name returns the source name
func (b *Bing) Name() string {
	return "bing"
}

// Run queries Bing for subdomains
func (b *Bing) Run(domain string, session *types.Session) ([]string, error) {
	var allSubdomains []string
	seen := make(map[string]bool)

	// Search multiple pages
	for page := 1; page <= 3; page++ {
		offset := (page - 1) * 10
		url := fmt.Sprintf("https://www.bing.com/search?q=site:%s&first=%d&count=10", domain, offset+1)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		resp, err := session.DoWithRetry(req)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		// Extract URLs from the search results
		urlRegex := regexp.MustCompile(`(?:https?://)([a-zA-Z0-9][-a-zA-Z0-9]*\.` + regexp.QuoteMeta(domain) + `)`)
		matches := urlRegex.FindAllStringSubmatch(string(body), -1)

		for _, match := range matches {
			if len(match) > 1 {
				sub := strings.ToLower(match[1])
				if !seen[sub] {
					seen[sub] = true
					allSubdomains = append(allSubdomains, sub)
				}
			}
		}

		// If no results on this page, stop paginating
		if len(matches) == 0 {
			break
		}
	}

	return allSubdomains, nil
}
