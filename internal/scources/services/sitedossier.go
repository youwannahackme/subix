package services

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// SiteDossier queries sitedossier.com for subdomains
type SiteDossier struct{}

// Name returns the source name
func (s *SiteDossier) Name() string {
	return "sitedossier"
}

// Run queries SiteDossier for subdomains across multiple pages
func (s *SiteDossier) Run(domain string, session *types.Session) ([]string, error) {
	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	for pageNum := 1; pageNum <= 1000; pageNum += 100 {
		url := fmt.Sprintf("http://www.sitedossier.com/parentdomain/%s/%d", domain, pageNum)
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

		if len(matches) == 0 || !strings.Contains(bodyStr, "Show next 100 items") {
			break
		}
	}

	return result, nil
}
