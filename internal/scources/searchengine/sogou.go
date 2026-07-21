package searchengine

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Sogou queries www.sogou.com search engine for subdomains
type Sogou struct{}

// Name returns the source name
func (s *Sogou) Name() string {
	return "sogou"
}

// Run queries Sogou search engine for subdomains across multiple pages
func (s *Sogou) Run(domain string, session *types.Session) ([]string, error) {
	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	for page := 1; page <= 20; page++ {
		url := fmt.Sprintf("https://www.sogou.com/web?query=site:.%s&page=%d", domain, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			break
		}
		req.Header.Set("User-Agent", types.DefaultUserAgent)

		resp, err := session.DoWithRetry(req)
		if err != nil {
			break
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			break
		}

		bodyStr := string(body)
		matches := re.FindAllString(bodyStr, -1)
		for _, sub := range matches {
			sub = strings.ToLower(strings.TrimSpace(sub))
			sub = strings.TrimPrefix(sub, ".")
			sub = strings.TrimPrefix(sub, "*.")
			if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
				seen[sub] = true
				result = append(result, sub)
			}
		}

		if len(matches) == 0 || !strings.Contains(bodyStr, `<a id="sogou_next"`) {
			break
		}
	}

	return result, nil
}
