package utils

import (
	"net"
	"strings"
)

// IsSubdomain checks if a host is a subdomain of the given domain
func IsSubdomain(host, domain string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	domain = strings.ToLower(strings.TrimSpace(domain))

	if host == domain {
		return false
	}

	return strings.HasSuffix(host, "."+domain)
}

// ExtractDomain extracts the root domain from a subdomain
func ExtractDomain(subdomain string) string {
	// Simple TLD stripping — not perfect but good enough for filtering
	parts := strings.Split(subdomain, ".")
	if len(parts) <= 2 {
		return subdomain
	}

	// Known TLDs that have two parts (co.uk, com.au, etc.)
	twoPartTLDs := map[string]bool{
		"co.uk": true, "com.au": true, "co.nz": true, "co.za": true,
		"co.in": true, "co.jp": true, "co.br": true, "co.il": true,
		"org.uk": true, "net.au": true, "ac.uk": true, "gov.uk": true,
		"edu.au": true, "ne.jp": true, "or.jp": true, "go.jp": true,
	}

	// Check for two-part TLD
	if len(parts) >= 3 {
		possibleTLD := parts[len(parts)-2] + "." + parts[len(parts)-1]
		if twoPartTLDs[possibleTLD] {
			if len(parts) >= 4 {
				return strings.Join(parts[len(parts)-3:], ".")
			}
			return subdomain
		}
	}

	// Default: last two parts are the domain
	return strings.Join(parts[len(parts)-2:], ".")
}

// SanitizeSubdomain cleans up a subdomain string
func SanitizeSubdomain(sub string) string {
	sub = strings.ToLower(strings.TrimSpace(sub))
	sub = strings.TrimPrefix(sub, ".")
	sub = strings.TrimSuffix(sub, ".")
	sub = strings.TrimPrefix(sub, "*.")
	// Remove trailing dots from FQDN
	for strings.HasSuffix(sub, ".") {
		sub = strings.TrimSuffix(sub, ".")
	}
	return sub
}

// ResolveHost resolves a hostname to IPs
func ResolveHost(host string) []string {
	ips, err := net.LookupHost(host)
	if err != nil {
		return nil
	}
	return ips
}
