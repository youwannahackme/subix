package services

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// NetCraft queries searchdns.netcraft.com for subdomains
type NetCraft struct{}

// Name returns the source name
func (n *NetCraft) Name() string {
	return "netcraft"
}

// Run queries NetCraft for subdomains across multiple pages
func (n *NetCraft) Run(domain string, session *types.Session) ([]string, error) {
	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	for page := 1; page <= 500; page += 20 {
		url := fmt.Sprintf("https://searchdns.netcraft.com/?restriction=site+contains&position=limited&host=*.%s&from=%d", domain, page)
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

		if !strings.Contains(bodyStr, "Next Page") || len(matches) == 0 {
			break
		}
	}

	return result, nil
}
