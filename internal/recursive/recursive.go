package recursive

import (
	"fmt"
	"strings"
	"sync"
)

// Engine handles recursive subdomain enumeration
type Engine struct {
	maxDepth int
	callback func(subdomain string, depth int)
	visited  sync.Map
}

// NewEngine creates a new recursive enumeration engine
func NewEngine(maxDepth int, callback func(string, int)) *Engine {
	return &Engine{
		maxDepth: maxDepth,
		callback: callback,
	}
}

// Run starts recursive enumeration on found subdomains
func (e *Engine) Run(rootDomain string, foundSubs []string) {
	// Group subdomains by their parent domain
	parentMap := make(map[string][]string)

	for _, sub := range foundSubs {
		parent := e.extractParent(sub, rootDomain)
		if parent != "" && parent != rootDomain {
			parentMap[parent] = append(parentMap[parent], sub)
		}
	}

	// For each parent domain that has multiple children, enumerate the parent
	for parent, children := range parentMap {
		// Only recurse if the parent looks like it could have more subdomains
		if len(children) >= 1 && e.getDepth(parent, rootDomain) < e.maxDepth {
			// Check if already visited
			if _, loaded := e.visited.LoadOrStore(parent, true); !loaded {
				fmt.Printf("    \033[35m↻ Recursing into: %s (%d known children)\033[0m\n", parent, len(children))
				e.callback(parent, e.getDepth(parent, rootDomain))
			}
		}
	}
}

// extractParent gets the parent domain of a subdomain relative to root
func (e *Engine) extractParent(subdomain, rootDomain string) string {
	if !strings.HasSuffix(subdomain, "."+rootDomain) {
		return ""
	}

	// Remove root domain
	remainder := strings.TrimSuffix(subdomain, "."+rootDomain)
	parts := strings.Split(remainder, ".")

	if len(parts) <= 1 {
		return rootDomain
	}

	// Return parent (everything except the first label + root)
	parent := strings.Join(parts[1:], ".") + "." + rootDomain
	return parent
}

// getDepth calculates how many levels deep a subdomain is
func (e *Engine) getDepth(subdomain, rootDomain string) int {
	if subdomain == rootDomain {
		return 0
	}

	remainder := strings.TrimSuffix(subdomain, "."+rootDomain)
	parts := strings.Split(remainder, ".")
	return len(parts) - 1
}
