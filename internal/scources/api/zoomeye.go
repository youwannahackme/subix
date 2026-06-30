package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// ZoomEye queries api.zoomeye.org API
type ZoomEye struct{}

// Name returns the source name
func (z *ZoomEye) Name() string {
	return "zoomeye"
}

type zoomeyeResponse struct {
	List []struct {
		Name string `json:"name"`
	} `json:"list"`
}

// Run queries ZoomEye for subdomains
func (z *ZoomEye) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["zoomeye"]
	if apiKey == "" {
		return nil, fmt.Errorf("zoomeye requires API key")
	}

	url := fmt.Sprintf("https://api.zoomeye.org/web/search?q=domain:%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "JWT "+apiKey)
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zoomeye status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data zoomeyeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, entry := range data.List {
		sub := strings.ToLower(strings.TrimSpace(entry.Name))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
