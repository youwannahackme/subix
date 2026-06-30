package services

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// HackerTarget queries HackerTarget API
type HackerTarget struct{}

// Name returns the source name
func (h *HackerTarget) Name() string {
	return "hackertarget"
}

// Run queries HackerTarget for subdomains
func (h *HackerTarget) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://api.hackertarget.com/hostsearch/?q=%s", domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", types.DefaultUserAgent)

	resp, err := session.DoWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hackertarget returned status %d", resp.StatusCode)
	}

	var subdomains []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, ",")
		if len(parts) > 0 {
			sub := strings.ToLower(strings.TrimSpace(parts[0]))
			if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
				seen[sub] = true
				subdomains = append(subdomains, sub)
			}
		}
	}

	return subdomains, scanner.Err()
}
