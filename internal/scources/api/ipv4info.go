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

// IPv4Info queries ipv4info.com API for subdomains
type IPv4Info struct{}

// Name returns the source name
func (i *IPv4Info) Name() string {
	return "ipv4info"
}

type ipv4infoResponse struct {
	Subdomains []string `json:"Subdomains"`
}

// Run queries IPv4Info API for subdomains across multiple pages
func (i *IPv4Info) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["ipv4info"]
	if apiKey == "" {
		return nil, fmt.Errorf("ipv4info requires API token")
	}

	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	for page := 0; page < 50; page++ {
		url := fmt.Sprintf("http://ipv4info.com/api_v1/?type=SUBDOMAINS&key=%s&value=%s&page=%d", apiKey, domain, page)
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

		var apiResp ipv4infoResponse
		if err := json.Unmarshal(body, &apiResp); err != nil || len(apiResp.Subdomains) < 300 || len(matches) == 0 {
			break
		}
	}

	return result, nil
}
