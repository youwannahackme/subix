package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Fofa queries fofa.info API
type Fofa struct{}

// Name returns the source name
func (f *Fofa) Name() string {
	return "fofa"
}

type fofaResponse struct {
	Results [][]string `json:"results"`
}

// Run queries Fofa for subdomains
func (f *Fofa) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["fofa"]
	if apiKey == "" {
		return nil, fmt.Errorf("fofa requires API key")
	}

	// FOFA requires base64 encoded query
	qbase64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("domain=\"%s\"", domain)))
	url := fmt.Sprintf("https://fofa.info/api/v1/search/all?qbase64=%s&size=10000&fields=host&key=%s", qbase64, apiKey)

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
		return nil, fmt.Errorf("fofa status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data fofaResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, entry := range data.Results {
		if len(entry) < 1 {
			continue
		}
		host := entry[0]
		// FOFA can return http:// prefix, clean it
		host = strings.Replace(host, "http://", "", 1)
		host = strings.Replace(host, "https://", "", 1)
		host = strings.Split(host, ":")[0] // remove port if any

		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" && strings.HasSuffix(host, "."+domain) && !seen[host] {
			seen[host] = true
			result = append(result, host)
		}
	}
	return result, nil
}
