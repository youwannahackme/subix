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

// Cloudflare queries Cloudflare zones and DNS records API for subdomains (read-only)
type Cloudflare struct{}

// Name returns the source name
func (c *Cloudflare) Name() string {
	return "cloudflare"
}

type cloudflareZonesResponse struct {
	Success bool `json:"success"`
	Result  []struct {
		ID string `json:"id"`
	} `json:"result"`
}

type cloudflareDNSRecordsResponse struct {
	Success    bool `json:"success"`
	ResultInfo struct {
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

// Run queries Cloudflare API for subdomains
func (c *Cloudflare) Run(domain string, session *types.Session) ([]string, error) {
	apiKey := session.Config.ProviderConfig.APIKeys["cloudflare"]
	if apiKey == "" {
		return nil, fmt.Errorf("cloudflare requires API token")
	}

	// 1. Get Zone ID for domain
	zoneURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", domain)
	req, err := http.NewRequest("GET", zoneURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var zonesResp cloudflareZonesResponse
	if err := json.Unmarshal(body, &zonesResp); err != nil || !zonesResp.Success || len(zonesResp.Result) == 0 {
		return nil, nil // zone not found in account
	}

	zoneID := zonesResp.Result[0].ID
	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	// 2. Paginate DNS records
	for page := 1; page <= 100; page++ {
		dnsURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?page=%d&per_page=100", zoneID, page)
		dnsReq, err := http.NewRequest("GET", dnsURL, nil)
		if err != nil {
			break
		}
		dnsReq.Header.Set("Authorization", "Bearer "+apiKey)
		dnsReq.Header.Set("Content-Type", "application/json")
		dnsReq.Header.Set("User-Agent", types.DefaultUserAgent)

		dnsResp, err := session.DoWithRetry(dnsReq)
		if err != nil {
			break
		}

		dnsBody, err := io.ReadAll(dnsResp.Body)
		dnsResp.Body.Close()
		if err != nil {
			break
		}

		matches := re.FindAllString(string(dnsBody), -1)
		for _, sub := range matches {
			sub = strings.ToLower(strings.TrimSpace(sub))
			sub = strings.TrimPrefix(sub, ".")
			sub = strings.TrimPrefix(sub, "*.")
			if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
				seen[sub] = true
				result = append(result, sub)
			}
		}

		var recordsResp cloudflareDNSRecordsResponse
		if err := json.Unmarshal(dnsBody, &recordsResp); err != nil || page >= recordsResp.ResultInfo.TotalPages {
			break
		}
	}

	return result, nil
}
