package certtransparency

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// Google queries Google's HTTPS transparency report for certificate subdomains
type Google struct{}

// Name returns the source name
func (g *Google) Name() string {
	return "google"
}

// Run queries Google CT search for subdomains
func (g *Google) Run(domain string, session *types.Session) ([]string, error) {
	url := fmt.Sprintf("https://transparencyreport.google.com/transparencyreport/api/v3/httpsreport/ct/certsearch?include_expired=true&include_subdomains=true&domain=%s", domain)

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
		return nil, fmt.Errorf("google certsearch status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Google CT search returns JS/JSON with a leading )]}'\n.
	// We extract subdomains using a regex matching any subdomain pattern ending with the target domain.
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))
	matches := re.FindAllString(string(body), -1)

	var subdomains []string
	seen := make(map[string]bool)

	for _, sub := range matches {
		sub = strings.ToLower(strings.TrimSpace(sub))
		sub = strings.TrimPrefix(sub, ".")
		sub = strings.TrimPrefix(sub, "*.")
		if sub != "" && strings.HasSuffix(sub, "."+domain) && !seen[sub] {
			seen[sub] = true
			subdomains = append(subdomains, sub)
		}
	}

	return subdomains, nil
}
