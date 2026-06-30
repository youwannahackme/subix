package searchengine

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/youwannahackme/subix/pkg/types"
)

// Google scrapes Google search results
type Google struct{}

// Name returns the source name
func (g *Google) Name() string {
	return "google"
}

var googleUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
}

// Run queries Google for subdomains
func (g *Google) Run(domain string, session *types.Session) ([]string, error) {
	// Add a small randomized delay to prevent instant bot detection
	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	url := fmt.Sprintf("https://www.google.com/search?q=site:*.%s&num=100", domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Rotate User-Agent
	ua := googleUserAgents[rand.Intn(len(googleUserAgents))]
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", "https://www.google.com/")

	// Create a custom transport that disables HTTP/2 to prevent "http2: response body closed" errors
	transport := &http.Transport{
		TLSNextProto: make(map[string]func(authority string, sha *tls.Conn) http.RoundTripper),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   session.Client.Timeout,
	}

	// Retry loop specifically with our HTTP/1.1 client, including reading the body
	var resp *http.Response
	var body []byte
	for i := 0; i < 3; i++ {
		resp, err = client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				body, err = io.ReadAll(resp.Body)
				resp.Body.Close()
				if err == nil {
					break
				}
				// If reading the body failed (e.g., connection closed mid-stream), retry
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusTooManyRequests {
				time.Sleep(time.Duration(i+1) * 3 * time.Second)
				continue
			}
			return nil, fmt.Errorf("google returned status %d", resp.StatusCode)
		}
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}

	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`(?i)[a-z0-9-]+\.` + regexp.QuoteMeta(domain))
	matches := re.FindAllString(string(body), -1)

	var result []string
	seen := make(map[string]bool)
	for _, sub := range matches {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" && !seen[sub] {
			seen[sub] = true
			result = append(result, sub)
		}
	}
	return result, nil
}
