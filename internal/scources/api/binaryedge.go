package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// BinaryEdge queries api.binaryedge.io API
type BinaryEdge struct{}

// Name returns the source name
func (b *BinaryEdge) Name() string {
	return "binaryedge"
}

type binaryedgeResponse struct {
	Events []string `json:"events"`
}

// Run queries BinaryEdge for subdomains
func (b *BinaryEdge) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["binaryedge"]
	if apiKey == "" {
		return nil, fmt.Errorf("binaryedge requires API key")
	}

	url := fmt.Sprintf("https://api.binaryedge.io/v2/query/domains/subdomains/%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Key", apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binaryedge status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data binaryedgeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, sub := range data.Events {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
