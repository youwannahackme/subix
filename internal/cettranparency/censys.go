package certtransparency

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Censys queries Censys certificate search API
type Censys struct{}

// Name returns the source name
func (c *Censys) Name() string {
	return "censys"
}

// censysResponse represents Censys API response
type censysResponse struct {
	Result struct {
		Hits []struct {
			Parsed struct {
				Names []string `json:"names"`
			} `json:"parsed"`
		} `json:"hits"`
	} `json:"result"`
}

// Run queries Censys for certificate subdomains
func (c *Censys) Run(domain string, session *types.Session) ([]string, error) {
	apiID := session.Config.ProviderConfig.APIKeys["censys_id"]
	apiSecret := session.Config.ProviderConfig.APIKeys["censys_secret"]
	if apiID == "" || apiSecret == "" {
		return nil, fmt.Errorf("censys requires API credentials")
	}

	url := fmt.Sprintf("https://search.censys.io/api/v2/certificates/search?q=parsed.names:%%20%s", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(apiID, apiSecret)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("censys status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result censysResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var subdomains []string
	seen := make(map[string]bool)

	for _, hit := range result.Result.Hits {
		for _, name := range hit.Parsed.Names {
			name = strings.TrimSpace(name)
			name = strings.TrimPrefix(name, "*.")
			if name != "" && strings.HasSuffix(name, "."+domain) && !seen[name] {
				seen[name] = true
				subdomains = append(subdomains, name)
			}
		}
	}

	return subdomains, nil
}
