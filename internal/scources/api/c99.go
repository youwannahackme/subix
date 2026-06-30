package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// C99 queries c99.nl API
type C99 struct{}

// Name returns the source name
func (c *C99) Name() string {
	return "c99"
}

type c99Response struct {
	Subdomains []struct {
		Subdomain string `json:"subdomain"`
	} `json:"subdomains"`
}

// Run queries C99 for subdomains
func (c *C99) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["c99"]
	if apiKey == "" {
		return nil, fmt.Errorf("c99 requires API key")
	}

	url := fmt.Sprintf("https://c99.nl/api.php?key=%s&action=subdomainfinder&domain=%s&output=json", apiKey, domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("c99 status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data c99Response
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, entry := range data.Subdomains {
		sub := strings.ToLower(strings.TrimSpace(entry.Subdomain))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
