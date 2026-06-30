package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Leakix queries leakix.net API
type Leakix struct{}

// Name returns the source name
func (l *Leakix) Name() string {
	return "leakix"
}

type leakixEntry struct {
	Hostname string `json:"hostname"`
}

// Run queries Leakix for subdomains
func (l *Leakix) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["leakix"]

	url := fmt.Sprintf("https://leakix.net/search?scope=leak&q=domain:%s", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("api-key", apiKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leakix status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data []leakixEntry
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var result []string
	seen := make(map[string]bool)
	for _, entry := range data {
		sub := strings.ToLower(strings.TrimSpace(entry.Hostname))
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
