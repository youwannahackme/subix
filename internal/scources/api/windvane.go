package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Windvane queries windvane.lichoin.com RPC service for subdomains
type Windvane struct{}

// Name returns the source name
func (w *Windvane) Name() string {
	return "windvane"
}

type windvaneRequest struct {
	Domain      string `json:"domain"`
	PageRequest struct {
		Page  int `json:"page"`
		Count int `json:"count"`
	} `json:"page_request"`
}

type windvaneResponse struct {
	Code int `json:"code"`
	Data struct {
		PageResponse struct {
			TotalPage int `json:"total_page"`
		} `json:"page_response"`
	} `json:"data"`
}

// Run queries Windvane for subdomains across multiple pages
func (w *Windvane) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["windvane"]

	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	page := 1
	totalPage := 1

	for page <= totalPage && page <= 50 {
		payload := windvaneRequest{
			Domain: domain,
		}
		payload.PageRequest.Page = page
		payload.PageRequest.Count = 1000

		jsonBody, err := json.Marshal(payload)
		if err != nil {
			break
		}

		req, err := http.NewRequest("POST", "https://windvane.lichoin.com/trpc.backendhub.public.WindvaneService/ListSubDomain", bytes.NewReader(jsonBody))
		if err != nil {
			break
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Referer", "https://windvane.lichoin.com")
		req.Header.Set("User-Agent", types.DefaultUserAgent)
		if apiKey != "" {
			req.Header.Set("X-Api-Key", apiKey)
		}

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

		var rpcResp windvaneResponse
		if err := json.Unmarshal(body, &rpcResp); err != nil || rpcResp.Code != 0 {
			break
		}
		if rpcResp.Data.PageResponse.TotalPage > totalPage {
			totalPage = rpcResp.Data.PageResponse.TotalPage
		}
		page++
	}

	return result, nil
}
