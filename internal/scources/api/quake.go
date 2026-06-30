package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Quake queries quake.360.net API
type Quake struct{}

// Name returns the source name
func (q *Quake) Name() string {
	return "quake"
}

type quakeResponse struct {
	Data []struct {
		Service struct {
			HTTP struct {
				Host string `json:"host"`
			} `json:"http"`
		} `json:"service"`
	} `json:"data"`
}

// Run queries Quake for subdomains
func (q *Quake) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["quake"]
	if apiKey == "" {
		return nil, fmt.Errorf("quake requires API key")
	}

	url := "https://quake.360.net/api/v3/search/quake_service"
	bodyMap := map[string]interface{}{
		"query": fmt.Sprintf("domain:\"*.%s\"", domain),
		"size":  100,
	}
	bodyBytes, _ := json.Marshal(bodyMap)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-QuakeToken", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("quake status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data quakeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, item := range data.Data {
		host := item.Service.HTTP.Host
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
