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

// Spyse queries api.spyse.com API for subdomains
type Spyse struct{}

// Name returns the source name
func (s *Spyse) Name() string {
	return "spyse"
}

type spyseResponse struct {
	Data struct {
		Items []json.RawMessage `json:"items"`
	} `json:"data"`
}

// Run queries Spyse API for subdomains across multiple offsets
func (s *Spyse) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["spyse"]
	if apiKey == "" {
		return nil, fmt.Errorf("spyse requires API token")
	}

	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	for offset := 0; offset <= 10000; offset += 100 {
		url := fmt.Sprintf("https://api.spyse.com/v3/data/domain/subdomain?domain=%s&offset=%d&limit=100", domain, offset)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			break
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
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

		var apiResp spyseResponse
		if err := json.Unmarshal(body, &apiResp); err != nil || len(apiResp.Data.Items) < 100 || len(matches) == 0 {
			break
		}
	}

	return result, nil
}
