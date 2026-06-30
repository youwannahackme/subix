package config

import (
	"os"
	"path/filepath"

	"github.com/youwannahackme/subix/pkg/types"
	"gopkg.in/yaml.v3"
)

// DefaultProviderConfig returns a config with all free sources enabled
func DefaultProviderConfig() *types.ProviderConfig {
	return &types.ProviderConfig{
		Sources: map[string]map[string]bool{
			"certtransparency": {
				"crtsh":       true,
				"censys":      false,
				"certspotter": true,
			},
			"services": {
				"dnsdumpster":     true,
				"hackertarget":    true,
				"urlscan":         true,
				"alienvault":      true,
				"anubis":          true,
				"subdomaincenter": true,
				"threatcrowd":     true,
				"columbus":        true,
				"jldc":            true,
				"sonar":           true,
				"robtex":          true,
				"rapiddns":        true,
				"synapsint":       true,
				"riddler":         true,
			},
			"webarchive": {
				"wayback":     true,
				"commoncrawl": true,
			},
			"searchengine": {
				"bing":       true,
				"duckduckgo": true,
				"google":     true,
				"yahoo":      true,
				"baidu":      true,
				"yandex":     true,
				"ask":        true,
			},
			"api": {
				"securitytrails": false,
				"virustotal":     false,
				"shodan":         false,
				"passivetotal":   false,
				"chaos":          false,
				"bevigil":        false,
				"zoomeye":        false,
				"fofa":           false,
				"hunter":         false,
				"intelx":         false,
				"leakix":         false,
				"netlas":         false,
				"binaryedge":     false,
				"threatbook":     false,
				"quake":          false,
				"c99":            false,
			},
		},
		APIKeys: map[string]string{
			"securitytrails": "",
			"virustotal":     "",
			"shodan":         "",
			"passivetotal":   "",
			"chaos":          "",
			"bevigil":        "",
			"zoomeye":        "",
			"fofa":           "",
			"hunter":         "",
			"intelx":         "",
			"leakix":         "",
			"netlas":         "",
			"binaryedge":     "",
			"threatbook":     "",
			"quake":          "",
			"c99":            "",
		},
		Censys: types.CensysConfig{
			ID:     "",
			Secret: "",
		},
	}
}

// LoadProviderConfig loads config from file, falls back to defaults
func LoadProviderConfig(path string) (*types.ProviderConfig, error) {
	if path == "" {
		// Try current directory configs/provider-config.yaml first
		cwdPath := filepath.Join("configs", "provider-config.yaml")
		if _, statErr := os.Stat(cwdPath); statErr == nil {
			path = cwdPath
		}
	}

	if path == "" {
		// Try default locations in home directory
		homeDir, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(homeDir, ".config", "subix", "provider-config.yaml")
			if _, statErr := os.Stat(defaultPath); statErr == nil {
				path = defaultPath
			}
		}
	}

	if path == "" {
		return DefaultProviderConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultProviderConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Merge API keys into the Censys struct if present
	if cfg.Censys.ID != "" {
		cfg.APIKeys["censys_id"] = cfg.Censys.ID
	}
	if cfg.Censys.Secret != "" {
		cfg.APIKeys["censys_secret"] = cfg.Censys.Secret
	}

	return cfg, nil
}
