package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/youwannahackme/subix/pkg/types"
)

// Intelx queries 2.intelx.io API
type Intelx struct{}

// Name returns the source name
func (i *Intelx) Name() string {
	return "intelx"
}

type intelxSearchResponse struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

type intelxResultResponse struct {
	Records []struct {
		Name string `json:"name"`
	} `json:"records"`
	Status int `json:"status"`
}

// Run queries Intelx for subdomains
func (i *Intelx) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["intelx"]
	if apiKey == "" {
		return nil, fmt.Errorf("intelx requires API key")
	}

	searchURL := "https://2.intelx.io/phonebook/search"
	bodyMap := map[string]interface{}{
		"term":       domain,
		"maxresults": 1000,
		"media":      0,
		"target":     3, // 3 = subdomains
	}
	bodyBytes, _ := json.Marshal(bodyMap)

	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("intelx search status %d", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var searchResp intelxSearchResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, err
	}

	resultURL := fmt.Sprintf("https://2.intelx.io/phonebook/search/result?id=%s&limit=1000", searchResp.ID)
	var result []string
	seen := make(map[string]bool)

	for attempt := 0; attempt < 3; attempt++ {
		time.Sleep(1 * time.Second)
		req2, err := http.NewRequest("GET", resultURL, nil)
		if err != nil {
			return nil, err
		}
		req2.Header.Set("x-key", apiKey)
		req2.Header.Set("User-Agent", types.DefaultUserAgent)

		resp2, err := session.Client.Do(req2)
		if err != nil {
			return nil, err
		}
		defer resp2.Body.Close()

		if resp2.StatusCode == http.StatusOK {
			respBody2, _ := io.ReadAll(resp2.Body)
			var resultResp intelxResultResponse
			if err := json.Unmarshal(respBody2, &resultResp); err == nil {
				for _, rec := range resultResp.Records {
					sub := strings.ToLower(strings.TrimSpace(rec.Name))
					if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
						seen[sub] = true
						result = append(result, sub)
					}
				}
				if resultResp.Status == 0 || len(result) > 0 {
					break
				}
			}
		}
	}

	return result, nil
}
