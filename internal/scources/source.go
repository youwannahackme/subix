package sources

import (
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

// NewHTTPClient creates a configured HTTP client from session config
func NewHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}
	return &http.Client{
		Transport: transport,
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
	}
}
