package sources

import (
	"net"
	"net/http"
	"time"

	"github.com/youwannahackme/subix/internal/cettranparency"
	"github.com/youwannahackme/subix/internal/scources/api"
	"github.com/youwannahackme/subix/internal/scources/searchengine"
	"github.com/youwannahackme/subix/internal/scources/services"
	"github.com/youwannahackme/subix/internal/scources/webarchive"
	"github.com/youwannahackme/subix/pkg/types"
)

// Source is the interface every enumeration source must implement
type Source interface {
	// Name returns the unique identifier of this source
	Name() string

	// Run executes enumeration for the given domain
	Run(domain string, session *types.Session) ([]string, error)
}

// BaseSource provides common functionality for all sources
type BaseSource struct {
	// empty base — sources embed this for future shared helpers
}

// rateLimitedTransport wraps an http.RoundTripper with rate limiting
type rateLimitedTransport struct {
	base    http.RoundTripper
	limiter types.RateLimiter
}

func (t *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.limiter != nil {
		if err := t.limiter.WaitContext(req.Context()); err != nil {
			return nil, err
		}
	}
	return t.base.RoundTrip(req)
}

// NewHTTPClient creates a configured HTTP client from session config
func NewHTTPClient(timeout time.Duration) *http.Client {
	return NewHTTPClientWithLimiter(timeout, nil)
}

// NewHTTPClientWithLimiter creates a configured HTTP client with rate limiting and transport isolation
func NewHTTPClientWithLimiter(timeout time.Duration, limiter types.RateLimiter) *http.Client {
	dialTimeout := 10 * time.Second
	if timeout > 0 && timeout < dialTimeout {
		dialTimeout = timeout
	}

	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}

	baseTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   dialTimeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
	}

	var rt http.RoundTripper = baseTransport
	if limiter != nil {
		rt = &rateLimitedTransport{
			base:    baseTransport,
			limiter: limiter,
		}
	}

	return &http.Client{
		Transport: rt,
		Timeout:   timeout,
	}
}

// AllSources returns the full list of registered sources
func AllSources() []Source {
	return []Source{
		// Certificate Transparency
		&certtransparency.Crtsh{},
		&certtransparency.Censys{},
		&certtransparency.Certspotter{},
		&certtransparency.Google{},
		// Services
		&services.DNSDumpster{},
		&services.HackerTarget{},
		&services.URLScan{},
		&services.AlienVault{},
		&services.Anubis{},
		&services.SubdomainCenter{},
		&services.ThreatCrowd{},
		&services.Columbus{},
		&services.JLDC{},
		&services.Sonar{},
		&services.Robtex{},
		&services.RapidDNS{},
		&services.Synapsint{},
		&services.Riddler{},
		&services.Sublist3r{},
		&services.ThreatMiner{},
		&services.MySSL{},
		&services.DNSGrep{},
		&services.CeBaidu{},
		&services.Chinaz{},
		&services.IP138{},
		&services.NetCraft{},
		&services.QianXun{},
		&services.SiteDossier{},
		// Web Archives
		&webarchive.Wayback{},
		&webarchive.CommonCrawl{},
		// Search Engines
		&searchengine.Bing{},
		&searchengine.DuckDuckGo{},
		&searchengine.Google{},
		&searchengine.Yahoo{},
		&searchengine.Baidu{},
		&searchengine.Yandex{},
		&searchengine.Ask{},
		&searchengine.Gitee{},
		&searchengine.So{},
		&searchengine.Sogou{},
		&searchengine.WzSearch{},
		// API sources
		&api.SecurityTrails{},
		&api.VirusTotal{},
		&api.Shodan{},
		&api.PassiveTotal{},
		&api.Chaos{},
		&api.BeVigil{},
		&api.ZoomEye{},
		&api.Fofa{},
		&api.Hunter{},
		&api.Intelx{},
		&api.Leakix{},
		&api.Netlas{},
		&api.BinaryEdge{},
		&api.ThreatBook{},
		&api.Quake{},
		&api.C99{},
		&api.FullHunt{},
		&api.Racent{},
		&api.ChinazAPI{},
		&api.Circl{},
		&api.Cloudflare{},
		&api.DNSDB{},
		&api.GitHub{},
		&api.IPv4Info{},
		&api.PassiveDNS{},
		&api.Spyse{},
		&api.Windvane{},
	}
}

