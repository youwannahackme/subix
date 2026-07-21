package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// GitHub queries GitHub code search API for subdomains
type GitHub struct{}

// Name returns the source name
func (g *GitHub) Name() string {
	return "github"
}

type githubSearchResponse struct {
	TotalCount int `json:"total_count"`
}

// Run queries GitHub API for subdomains across multiple pages
func (g *GitHub) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["github"]
	if apiKey == "" {
		return nil, fmt.Errorf("github requires API token")
	}

	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	for page := 1; page <= 10; page++ {
		url := fmt.Sprintf("https://api.github.com/search/code?q=%s&per_page=100&page=%d&sort=indexed", domain, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			break
		}
		req.Header.Set("Authorization", "token "+apiKey)
		req.Header.Set("Accept", "application/vnd.github.v3.text-match+json")
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

		var searchResp githubSearchResponse
		if err := json.Unmarshal(body, &searchResp); err != nil || page*100 >= searchResp.TotalCount || page*100 >= 1000 || len(matches) == 0 {
			break
		}
	}

	return result, nil
}
