package services

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// DNSDumpster queries dnsdumpster.com
type DNSDumpster struct{}

// Name returns the source name
func (d *DNSDumpster) Name() string {
	return "dnsdumpster"
}

// Run queries DNSDumpster for subdomains
func (d *DNSDumpster) Run(domain string, session *types.Session) ([]string, error) {
	// Step 1: GET the page to extract the JWT token
	getReq, err := http.NewRequest("GET", "https://dnsdumpster.com/", nil)
	if err != nil {
		return nil, err
	}
	getReq.Header.Set("User-Agent", types.DefaultUserAgent)

	getResp, err := session.DoWithRetry(getReq)
	if err != nil {
		return nil, err
	}
	defer getResp.Body.Close()

	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		return nil, err
	}

	bodyStr := string(getBody)

	// Extract the Authorization token from the HTML
	tokenRegex := regexp.MustCompile(`"Authorization":\s*"([^"]+)"`)
	matches := tokenRegex.FindStringSubmatch(bodyStr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not extract CSRF/Authorization token from dnsdumpster")
	}
	token := matches[1]

	// Step 2: POST to the HTMX endpoint with the target domain
	postURL := "https://api.dnsdumpster.com/htmld/"
	postData := fmt.Sprintf("target=%s", domain)

	postReq, err := http.NewRequest("POST", postURL, strings.NewReader(postData))
	if err != nil {
		return nil, err
	}
	postReq.Header.Set("User-Agent", types.DefaultUserAgent)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Referer", "https://dnsdumpster.com/")
	postReq.Header.Set("Authorization", token)
	postReq.Header.Set("HX-Request", "true")

	// Copy cookies from GET response
	for _, cookie := range getResp.Cookies() {
		postReq.AddCookie(cookie)
	}

	postResp, err := session.DoWithRetry(postReq)
	if err != nil {
		return nil, err
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dnsdumpster returned status %d", postResp.StatusCode)
	}

	postBody, err := io.ReadAll(postResp.Body)
	if err != nil {
		return nil, err
	}

	// Parse subdomains from the HTML response
	rowRegex := regexp.MustCompile(`<td[^>]*>([^<]+\.` + regexp.QuoteMeta(domain) + `)</td>`)
	tableMatches := rowRegex.FindAllStringSubmatch(string(postBody), -1)

	var subdomains []string
	seen := make(map[string]bool)

	for _, match := range tableMatches {
		sub := strings.TrimSpace(match[1])
		sub = strings.ToLower(sub)
		if sub != "" && sub != domain && !seen[sub] {
			seen[sub] = true
			subdomains = append(subdomains, sub)
		}
	}

	return subdomains, nil
}
