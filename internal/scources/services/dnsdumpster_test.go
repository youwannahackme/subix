package services

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/youwannahackme/subix/pkg/types"
)

func TestDNSDumpster(t *testing.T) {
	domain := "riphahfsd.edu.pk"
	cfg := &types.Config{
		Timeout: 15 * time.Second,
	}
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	// Step 1: GET the home page to extract the JWT token
	req, err := http.NewRequest("GET", "https://dnsdumpster.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	bodyStr := string(bodyBytes)

	// Extract the Authorization token
	// Format: hx-headers='{"Authorization": "eyJ..."}' or similar
	tokenRegex := regexp.MustCompile(`"Authorization":\s*"([^"]+)"`)
	matches := tokenRegex.FindStringSubmatch(bodyStr)
	if len(matches) < 2 {
		t.Fatal("Could not extract Authorization token from page")
	}
	token := matches[1]
	t.Logf("Extracted token: %s...", token[:30])

	// Step 2: POST to the HTMX endpoint
	postURL := "https://api.dnsdumpster.com/htmld/"
	postData := fmt.Sprintf("target=%s", domain)

	postReq, err := http.NewRequest("POST", postURL, strings.NewReader(postData))
	if err != nil {
		t.Fatal(err)
	}
	postReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Referer", "https://dnsdumpster.com/")
	postReq.Header.Set("Authorization", token)
	postReq.Header.Set("HX-Request", "true")

	// Copy cookies from GET response just in case
	for _, cookie := range resp.Cookies() {
		postReq.AddCookie(cookie)
	}

	postResp, err := client.Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	defer postResp.Body.Close()

	t.Logf("POST Status: %s", postResp.Status)

	postBodyBytes, err := io.ReadAll(postResp.Body)
	if err != nil {
		t.Fatal(err)
	}

	postBodyStr := string(postBodyBytes)
	t.Logf("POST Body length: %d", len(postBodyStr))

	// Parse subdomains
	rowRegex := regexp.MustCompile(`<td[^>]*>([^<]+\.` + regexp.QuoteMeta(domain) + `)</td>`)
	tableMatches := rowRegex.FindAllStringSubmatch(postBodyStr, -1)

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

	t.Logf("Found %d subdomains: %v", len(subdomains), subdomains)
	if len(subdomains) == 0 {
		// Print a snippet of the body if no subdomains found
		if len(postBodyStr) > 500 {
			t.Logf("Snippet: %s", postBodyStr[:500])
		} else {
			t.Logf("Body: %s", postBodyStr)
		}
		t.Fatal("No subdomains found")
	}
}
