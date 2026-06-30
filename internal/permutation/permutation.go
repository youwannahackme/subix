package permutation

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// DefaultWordlist returns the built-in permutation wordlist
func DefaultWordlist() []string {
	return []string{
		// Environments
		"dev", "development", "staging", "stage", "test", "testing",
		"prod", "production", "uat", "sit", "qa", "ci", "cd",
		"demo", "sandbox", "preview", "beta", "alpha", "gamma",
		// Infrastructure
		"api", "app", "admin", "portal", "dashboard", "panel",
		"console", "manage", "manager", "cms", "backend", "frontend",
		"web", "www", "m", "mobile", "desktop",
		// Services
		"mail", "email", "smtp", "imap", "pop", "ftp", "sftp",
		"vpn", "ssh", "rdp", "remote", "gateway", "proxy",
		"cdn", "static", "assets", "media", "images", "img",
		"files", "downloads", "upload", "uploads", "storage",
		// Databases
		"db", "database", "mysql", "postgres", "pgsql", "mongo",
		"redis", "elastic", "es", "cassandra", "couch",
		// Auth
		"auth", "login", "sso", "oauth", "identity", "id",
		"accounts", "account", "profile", "user", "users",
		// Monitoring
		"monitor", "monitoring", "grafana", "prometheus", "kibana",
		"log", "logs", "logging", "splunk", "datadog", "status",
		// Networking
		"ns1", "ns2", "ns3", "ns4", "dns", "dns1", "dns2",
		"mx", "mx1", "mx2", "relay", "smtp1", "smtp2",
		// Regions
		"us", "eu", "uk", "asia", "ap", "emea", "latam",
		"us-east", "us-west", "eu-west", "eu-central",
		"us1", "us2", "eu1", "eu2", "ap1", "ap2",
		// Cloud
		"aws", "gcp", "azure", "cloud", "s3", "bucket",
		"ec2", "lambda", "edge", "cf", "cloudfront",
		// Common prefixes
		"internal", "external", "public", "private", "corp",
		"new", "old", "v1", "v2", "v3", "legacy",
		"go", "get", "open", "api2", "api-v2", "rest",
		// Misc
		"blog", "docs", "doc", "wiki", "help", "support",
		"shop", "store", "pay", "billing", "checkout",
		"search", "find", "track", "analytics", "metrics",
		"notify", "notification", "push", "webhook", "hook",
		"cron", "job", "worker", "task", "queue", "broker",
		"cache", "memcached", "varnish", "nginx", "apache",
		"jenkins", "ci", "cd", "build", "deploy", "release",
		"git", "gitlab", "github", "repo", "code",
		"chat", "slack", "teams", "connect", "realtime",
		"video", "stream", "live", "radio", "tv",
		"news", "press", "marketing", "seo", "social",
		"partners", "affiliate", "referral", "invite",
	}
}

// LoadWordlist loads a wordlist from a file
func LoadWordlist(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var words []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			words = append(words, strings.ToLower(line))
		}
	}
	return words, nil
}

// Generate creates permuted subdomains from existing subdomains and a wordlist
func Generate(domain string, existingSubs []string, wordlist []string) []string {
	var permutations []string
	seen := make(map[string]bool)

	// Strategy 1: word.domain.com (direct prefix)
	for _, word := range wordlist {
		sub := fmt.Sprintf("%s.%s", word, domain)
		if !seen[sub] {
			seen[sub] = true
			permutations = append(permutations, sub)
		}
	}

	// Strategy 2: word.existing.domain.com (prepend to existing)
	for _, existing := range existingSubs {
		// Only use subdomains that are direct children or one level deeper
		parts := strings.Split(existing, ".")
		if len(parts) < 3 {
			continue
		}

		// Get the base part to prepend to
		base := strings.Join(parts[1:], ".")

		for _, word := range wordlist {
			sub := fmt.Sprintf("%s.%s", word, base)
			if !seen[sub] {
				seen[sub] = true
				permutations = append(permutations, sub)
			}
		}
	}

	// Strategy 3: existing-word.domain.com (append suffix to first label)
	for _, existing := range existingSubs {
		parts := strings.Split(existing, ".")
		if len(parts) < 2 {
			continue
		}

		base := strings.Join(parts[1:], ".")
		label := parts[0]

		for _, word := range wordlist {
			// label-word.base
			sub := fmt.Sprintf("%s-%s.%s", label, word, base)
			if !seen[sub] {
				seen[sub] = true
				permutations = append(permutations, sub)
			}
			// word-label.base
			sub2 := fmt.Sprintf("%s-%s.%s", word, label, base)
			if !seen[sub2] {
				seen[sub2] = true
				permutations = append(permutations, sub2)
			}
			// label_word.base
			sub3 := fmt.Sprintf("%s_%s.%s", label, word, base)
			if !seen[sub3] {
				seen[sub3] = true
				permutations = append(permutations, sub3)
			}
		}
	}

	// Strategy 4: number patterns (e.g., api1, api2, app1, app2)
	numberSuffixes := []string{"1", "2", "3", "01", "02", "03", "10", "11", "12"}
	numberPrefixes := []string{"app", "api", "web", "node", "srv", "server", "worker", "pod"}
	for _, prefix := range numberPrefixes {
		for _, num := range numberSuffixes {
			sub := fmt.Sprintf("%s%s.%s", prefix, num, domain)
			if !seen[sub] {
				seen[sub] = true
				permutations = append(permutations, sub)
			}
		}
	}

	return permutations
}

// mu is unused but prevents compiler complaints in some Go versions
var _ sync.Mutex
