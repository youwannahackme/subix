package services

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/youwannahackme/subix/pkg/types"
)

// QianXun queries www.dnsscan.cn for subdomains
type QianXun struct{}

// Name returns the source name
func (q *QianXun) Name() string {
	return "qianxun"
}

// Run queries QianXun for subdomains across multiple pages
func (q *QianXun) Run(domain string, session *types.Session) ([]string, error) {
	var result []string
	seen := make(map[string]bool)
	re := regexp.MustCompile(`(?i)[a-z0-9.-]+\.` + regexp.QuoteMeta(domain))

	for page := 1; page <= 50; page++ {
		reqURL := fmt.Sprintf("https://www.dnsscan.cn/dns.html?keywords=%s&page=%d", domain, page)
		formData := url.Values{
			"ecmsfrom": {""},
			"show":     {""},
			"num":      {""},
			"classid":  {"0"},
			"keywords": {domain},
		}

		req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
		if err != nil {
			break
		}
		req.Header.Set("User-Agent", types.DefaultUserAgent)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

		if len(matches) == 0 || !strings.Contains(bodyStr, `<div id="page" class="pagelist">`) || strings.Contains(bodyStr, `<li class="disabled"><span>&raquo;</span></li>`) {
			break
		}
	}

	return result, nil
}
