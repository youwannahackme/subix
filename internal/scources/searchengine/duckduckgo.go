package searchengine

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// DuckDuckGo queries DuckDuckGo HTML version for subdomains
type DuckDuckGo struct{}

// Name returns the source name
func (d *DuckDuckGo) Name() string {
	return "duckduckgo"
}

// Run queries DuckDuckGo for subdomains
func (d *DuckDuckGo) Run(domain string, session *types.Session) ([]string, error) {
	var allSubdomains []string
	seen := make(map[string]bool)

	for page := 1; page <= 3; page++ {
		url := fmt.Sprintf("https://html.duckduckgo.com/html/?q=site:%%25.%s&s=%d", domain, (page-1)*20)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		resp, err := session.DoWithRetry(req)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		// Extract URLs from result links
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

		if len(matches) == 0 {
			break
		}
	}

	return allSubdomains, nil
}
