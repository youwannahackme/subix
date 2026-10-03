package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/youwannahackme/subix/internal/output"
	"github.com/youwannahackme/subix/internal/permutation"
	"github.com/youwannahackme/subix/internal/recursive"
	"github.com/youwannahackme/subix/internal/resolution"
	"github.com/youwannahackme/subix/internal/scources"
	"github.com/youwannahackme/subix/internal/utils"
	"github.com/youwannahackme/subix/pkg/types"
)

// Runner orchestrates the entire subdomain enumeration process
type Runner struct {
	config    *types.Config
	domains   []string
	sources   []sources.Source
	results   chan *types.SubdomainResult
	stats     *types.Stats
	seen      sync.Map
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	wildcards *resolution.WildcardDetector
	resolver  *resolution.Resolver
	output    *output.Writer
	session   *types.Session
}

// NewRunner creates a new Runner instance
func NewRunner(cfg *types.Config, domains []string) (*Runner, error) {
	ctx, cancel := context.WithCancel(context.Background())

	allSources := sources.AllSources()
	filteredSources := filterSources(allSources, cfg)

	client := sources.NewHTTPClient(cfg.Timeout)
	session := &types.Session{
		Config: cfg,
		Client: client,
	}

	var resolver *resolution.Resolver
	if cfg.ResolveDNS || cfg.OnlyResolved || cfg.WildcardFilter {
		resolver = resolution.NewResolver(cfg.Threads)
	}

	var wildcardDetector *resolution.WildcardDetector
	if cfg.WildcardFilter {
		wildcardDetector = resolution.NewWildcardDetector(resolver)
	}

	writer, err := output.NewWriter(cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	return &Runner{
		config:    cfg,
		domains:   domains,
		sources:   filteredSources,
		results:   make(chan *types.SubdomainResult, 1000),
		stats:     types.NewStats(),
		ctx:       ctx,
		cancel:    cancel,
		wildcards: wildcardDetector,
		resolver:  resolver,
		output:    writer,
		session:   session,
	}, nil
}

// filterSources applies include/exclude filters to the source list
func filterSources(all []sources.Source, cfg *types.Config) []sources.Source {
	// Build a map of which sources are enabled in provider config
	enabledMap := make(map[string]bool)
	if cfg.ProviderConfig != nil && cfg.ProviderConfig.Sources != nil {
		for _, category := range cfg.ProviderConfig.Sources {
			for name, enabled := range category {
				if enabled || cfg.AllSources {
					enabledMap[name] = true
				}
			}
		}
	}

	var filtered []sources.Source
	for _, src := range all {
		name := src.Name()

		// Check include list
		if len(cfg.IncludeSources) > 0 {
			found := false
			for _, inc := range cfg.IncludeSources {
				if inc == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check exclude list
		if len(cfg.ExcludeSources) > 0 {
			excluded := false
			for _, exc := range cfg.ExcludeSources {
				if exc == name {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		// Check provider config (unless -all-sources or explicitly included)
		if !cfg.AllSources && len(cfg.IncludeSources) == 0 {
			if !enabledMap[name] {
				continue
			}
		}

		filtered = append(filtered, src)
	}

	return filtered
}

// Run executes the full enumeration pipeline
func (r *Runner) Run() error {
	startTime := time.Now()
	r.stats.Domains = len(r.domains)
	r.stats.SourcesUsed = len(r.sources)

	if !r.config.Silent {
		r.printBanner()
		fmt.Fprintf(os.Stderr, "\n  Domains:    %s\n", strings.Join(r.domains, ", "))
		fmt.Fprintf(os.Stderr, "  Sources:     %d\n", r.stats.SourcesUsed)
		fmt.Fprintf(os.Stderr, "  Threads:     %d\n", r.config.Threads)
		if r.config.Recursive {
			fmt.Fprintf(os.Stderr, "  Recursive:   depth=%d\n", r.config.MaxDepth)
		}
		if r.config.Permutation {
			fmt.Fprintf(os.Stderr, "  Permutation: enabled\n")
		}
		if r.config.ResolveDNS {
			fmt.Fprintf(os.Stderr, "  DNS Resolve: enabled\n")
		}
		if r.config.WildcardFilter {
			fmt.Fprintf(os.Stderr, "  Wildcard:    enabled\n")
		}
		fmt.Fprintln(os.Stderr)
	}

	// Start the output writer goroutine
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.output.Write(r.results, r.ctx)
	}()

	// Process each domain
	for _, domain := range r.domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}

		// Detect wildcards for this domain
		if r.wildcards != nil {
			if !r.config.Silent {
				fmt.Fprintf(os.Stderr, "  [%s] Detecting wildcards...\n", utils.ColorCyan(domain))
			}
			r.wildcards.Detect(domain)
		}

		// Run all sources
		r.enumerateDomain(domain, 0)

		// Recursive enumeration
		if r.config.Recursive {
			r.runRecursive(domain)
		}

		// Permutation
		if r.config.Permutation {
			r.runPermutation(domain)
		}
	}

	// Close results channel and wait for writer
	close(r.results)
	r.wg.Wait()

	r.stats.Duration = time.Since(startTime)

	if r.config.ShowStats && !r.config.Silent {
		r.printStats()
	}

	r.output.Close()

	return nil
}

// enumerateDomain runs all sources against a single domain
func (r *Runner) enumerateDomain(domain string, depth int) {
	if !r.config.Silent {
		indent := strings.Repeat("  ", depth+1)
		fmt.Fprintf(os.Stderr, "%s[%s] Enumerating with %d sources...\n", indent, utils.ColorCyan(domain), len(r.sources))
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, r.config.Threads)

	for _, src := range r.sources {
		wg.Add(1)
		go func(s sources.Source) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			select {
			case <-r.ctx.Done():
				return
			default:
			}

			var sourceLimiter *utils.RateLimiter
			if r.config.RateLimit > 0 {
				sourceLimiter = utils.NewRateLimiter(r.config.RateLimit)
			}
			sourceClient := sources.NewHTTPClientWithLimiter(r.config.Timeout, sourceLimiter)
			sourceSession := &types.Session{
				Config:      r.config,
				Client:      sourceClient,
				Context:     r.ctx,
				RateLimiter: sourceLimiter,
				SourceName:  s.Name(),
				Stats:       r.stats,
			}

			if sourceLimiter != nil {
				if err := sourceLimiter.WaitContext(r.ctx); err != nil {
					return
				}
			}

			subdomains, err := s.Run(domain, sourceSession)

			if err != nil {
				errStr := err.Error()
				isMissingKeys := strings.Contains(errStr, "requires API key") ||
					strings.Contains(errStr, "requires API credentials") ||
					strings.Contains(errStr, "requires API key and username")

				if isMissingKeys {
					if !r.config.Silent {
						fmt.Fprintf(os.Stderr, "%s  \033[33m! %s: skipped (missing API credentials)\033[0m\n", strings.Repeat("  ", depth+2), s.Name())
					}
					return
				}

				r.mu.Lock()
				r.stats.Errors[s.Name()]++
				r.mu.Unlock()
				if !r.config.Silent {
					fmt.Fprintf(os.Stderr, "%s  \033[31m✗ %s: %v\033[0m\n", strings.Repeat("  ", depth+2), s.Name(), err)
				}
				return
			}

			count := 0
			for _, sub := range subdomains {
				sub = strings.ToLower(strings.TrimSpace(sub))
				if sub == "" || sub == domain {
					continue
				}

				// Wildcard filtering
				if r.wildcards != nil && r.wildcards.IsWildcard(sub) {
					atomic.AddInt64(&r.stats.WildcardFilter, 1)
					continue
				}

				// Deduplication & Seen Tracking
				// Always populate r.seen regardless of RemoveDuplicate so that
				// recursive enumeration and permutation engines always have all found subdomains
				_, alreadySeen := r.seen.LoadOrStore(sub, true)
				if r.config.RemoveDuplicate && alreadySeen {
					continue
				}

				atomic.AddInt64(&r.stats.TotalFound, 1)

				result := &types.SubdomainResult{
					Host:   sub,
					Source: s.Name(),
				}

				// DNS resolution (reads from cache populated by wildcard check or performs and caches resolution)
				if r.resolver != nil {
					dnsDetails := r.resolver.ResolveDetails(sub)
					result.IPs = dnsDetails.IPs
					result.A = dnsDetails.A
					result.AAAA = dnsDetails.AAAA
					result.CNAME = dnsDetails.CNAME

					if len(dnsDetails.IPs) > 0 {
						atomic.AddInt64(&r.stats.Resolved, 1)
					}
					if r.config.OnlyResolved && len(dnsDetails.IPs) == 0 {
						continue
					}
				}

				count++
				r.results <- result
			}

			r.mu.Lock()
			r.stats.SourceCount[s.Name()] += count
			r.mu.Unlock()

			if !r.config.Silent && count > 0 {
				fmt.Fprintf(os.Stderr, "%s  \033[32m✓ %s: %d subdomains\033[0m\n", strings.Repeat("  ", depth+2), s.Name(), count)
			}
		}(src)
	}

	wg.Wait()
}

// runRecursive performs recursive subdomain enumeration
func (r *Runner) runRecursive(domain string) {
	if !r.config.Silent {
		fmt.Fprintf(os.Stderr, "\n  [%s] \033[35mStarting recursive enumeration (depth=%d)...\033[0m\n", utils.ColorCyan(domain), r.config.MaxDepth)
	}

	// Collect all found subdomains so far
	var foundSubs []string
	r.seen.Range(func(key, _ interface{}) bool {
		if s, ok := key.(string); ok {
			if strings.HasSuffix(s, "."+domain) && s != domain {
				foundSubs = append(foundSubs, s)
			}
		}
		return true
	})

	engine := recursive.NewEngine(r.config.MaxDepth, func(subDomain string, currentDepth int) {
		if currentDepth >= r.config.MaxDepth {
			return
		}
		r.enumerateDomain(subDomain, currentDepth+1)
	})

	engine.Run(domain, foundSubs)
}

// runPermutation generates and checks permuted subdomains
func (r *Runner) runPermutation(domain string) {
	if !r.config.Silent {
		fmt.Fprintf(os.Stderr, "\n  [%s] \033[33mStarting permutation engine...\033[0m\n", utils.ColorCyan(domain))
	}

	// Collect existing subdomains for base permutation
	var existingSubs []string
	r.seen.Range(func(key, _ interface{}) bool {
		if s, ok := key.(string); ok {
			existingSubs = append(existingSubs, s)
		}
		return true
	})

	wordlist := permutation.DefaultWordlist()
	if r.config.Wordlist != "" {
		custom, err := permutation.LoadWordlist(r.config.Wordlist)
		if err != nil {
			if !r.config.Silent {
				fmt.Fprintf(os.Stderr, "  \033[31m✗ Could not load wordlist: %v\033[0m\n", err)
			}
		} else {
			wordlist = custom
		}
	}

	perms := permutation.Generate(domain, existingSubs, wordlist)
	if !r.config.Silent {
		fmt.Fprintf(os.Stderr, "  Generated %d permutations\n", len(perms))
	}

	// Resolve permutations
	if r.resolver == nil {
		r.resolver = resolution.NewResolver(r.config.Threads)
	}

	semaphore := make(chan struct{}, r.config.Threads)
	var wg sync.WaitGroup
	found := int64(0)

	for _, perm := range perms {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			select {
			case <-r.ctx.Done():
				return
			default:
			}

			if _, loaded := r.seen.LoadOrStore(p, true); loaded {
				return
			}

			dnsDetails := r.resolver.ResolveDetails(p)
			if len(dnsDetails.IPs) > 0 {
				atomic.AddInt64(&found, 1)
				atomic.AddInt64(&r.stats.TotalFound, 1)
				atomic.AddInt64(&r.stats.Resolved, 1)

				r.results <- &types.SubdomainResult{
					Host:   p,
					Source: "permutation",
					IPs:    dnsDetails.IPs,
					A:      dnsDetails.A,
					AAAA:   dnsDetails.AAAA,
					CNAME:  dnsDetails.CNAME,
				}
			}
		}(perm)
	}

	wg.Wait()

	if !r.config.Silent {
		fmt.Fprintf(os.Stderr, "  \033[32m✓ Permutation found %d new subdomains\033[0m\n", found)
	}
}

// printBanner shows the tool banner
func (r *Runner) printBanner() {
	banner := `
    ███████╗██╗   ██╗██████╗ ██╗██╗  ██╗
    ██╔════╝██║   ██║██╔══██╗██║╚██╗██╔╝
    ███████╗██║   ██║██████╔╝██║ ╚███╔╝ 
    ╚════██║██║   ██║██╔══██╗██║ ██╔██╗ 
    ███████║╚██████╔╝██████╔╝██║██╔╝ ██╗
    ╚══════╝ ╚═════╝ ╚═════╝ ╚═╝╚═╝  ╚═╝
`
	fmt.Fprintf(os.Stderr, "\033[36m%s\033[0m", banner)
	fmt.Fprintf(os.Stderr, "  \033[33mSUBIX: High-Impact Subdomain Enumeration Engine v2.0\033[0m\n")
}

// printStats shows final statistics
func (r *Runner) printStats() {
	uniqueCount := int64(0)
	r.seen.Range(func(_, _ interface{}) bool {
		uniqueCount++
		return true
	})
	r.stats.UniqueSubs = uniqueCount

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  ──────────────────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  \033[36mStatistics:\033[0m\n")
	fmt.Fprintf(os.Stderr, "  ──────────────────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  Domains enumerated:    %d\n", r.stats.Domains)
	fmt.Fprintf(os.Stderr, "  Sources used:          %d\n", r.stats.SourcesUsed)
	fmt.Fprintf(os.Stderr, "  Total results:         %d\n", r.stats.TotalFound)
	fmt.Fprintf(os.Stderr, "  Unique subdomains:     %d\n", r.stats.UniqueSubs)
	fmt.Fprintf(os.Stderr, "  Resolved:              %d\n", r.stats.Resolved)
	fmt.Fprintf(os.Stderr, "  Wildcards filtered:    %d\n", r.stats.WildcardFilter)
	fmt.Fprintf(os.Stderr, "  Duration:              %s\n", r.stats.Duration.Round(time.Millisecond))

	if len(r.stats.SourceCount) > 0 || len(r.stats.RateLimited) > 0 || len(r.stats.Blocked) > 0 {
		fmt.Fprintf(os.Stderr, "\n  \033[36mPer-Source Breakdown:\033[0m\n")
		for _, src := range r.sources {
			name := src.Name()
			count := r.stats.SourceCount[name]
			rateLimits := r.stats.RateLimited[name]
			blocked := r.stats.Blocked[name]

			extra := ""
			if rateLimits > 0 {
				extra = fmt.Sprintf(" \033[33m[Rate-limited: %d]\033[0m", rateLimits)
			}
			if blocked > 0 {
				extra += fmt.Sprintf(" \033[31m[Blocked/Auth: %d]\033[0m", blocked)
			}

			if count > 0 {
				bar := strings.Repeat("█", min(count/5, 30))
				fmt.Fprintf(os.Stderr, "    %-18s %4d  %s%s\n", name, count, utils.ColorGreen(bar), extra)
			} else if extra != "" {
				fmt.Fprintf(os.Stderr, "    %-18s    0  %s\n", name, extra)
			}
		}
	}

	if len(r.stats.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "\n  \033[31mErrors:\033[0m\n")
		for name, count := range r.stats.Errors {
			fmt.Fprintf(os.Stderr, "    %-18s %d\n", name, count)
		}
	}

	fmt.Fprintf(os.Stderr, "  ──────────────────────────────────────\n\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
