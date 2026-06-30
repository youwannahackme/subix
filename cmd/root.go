package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/youwannahackme/subix/internal/config"
	"github.com/youwannahackme/subix/internal/runner"
	"github.com/youwannahackme/subix/pkg/types"
)

var (
	version            = "2.0.0"
	domain             string
	domainList         string
	outputFile         string
	jsonOutput         bool
	csvOutput          bool
	threads            int
	timeout            int
	recursive          bool
	depth              int
	rateLimit          int
	resolveDNS         bool
	wildcardFilter     bool
	permute            bool
	wordlistPath       string
	sourcesList        string
	excludeSourcesList string
	allSources         bool
	verbose            bool
	showStats          bool
	showSources        bool
	configPath         string
	onlyResolved       bool
	removeDuplicate    bool
)

var rootCmd = &cobra.Command{
	Use:   "subix",
	Short: "🔱 🔱 SUBIX: High-Impact Subdomain Enumeration Engine",
	Long: `
    ███████╗██╗   ██╗██████╗ ██╗██╗  ██╗
    ██╔════╝██║   ██║██╔══██╗██║╚██╗██╔╝
    ███████╗██║   ██║██████╔╝██║ ╚███╔╝ 
    ╚════██║██║   ██║██╔══██╗██║ ██╔██╗ 
    ███████║╚██████╔╝██████╔╝██║██╔╝ ██╗
    ╚══════╝ ╚═════╝ ╚═════╝ ╚═╝╚═╝  ╚═╝

  🔱 SUBIX: High-Impact Subdomain Enumeration Engine

Subix combines 16+ passive OSINT sources, recursive enumeration,
permutation engine, wildcard detection, and DNS resolution into
a single blazing-fast tool.

Examples:
  subix -d example.com
  subix -d example.com -o results.txt -recursive -depth 2
  subix -d example.com -resolve -wildcard -permute
  subix -l domains.txt -t 50 -j -all-sources
  subix -d example.com -sources crtsh,hackertarget,anubis
  subix --list-sources
`,
	Run: func(cmd *cobra.Command, args []string) {
		if showSources {
			printAvailableSources()
			return
		}

		if domain == "" && domainList == "" {
			fmt.Fprintf(os.Stderr, "Error: provide -d or -l flag\n")
			_ = cmd.Help()
			os.Exit(1)
		}

		cfg := &types.Config{
			Threads:         threads,
			Timeout:         time.Duration(timeout) * time.Second,
			MaxDepth:        depth,
			RateLimit:       rateLimit,
			ResolveDNS:      resolveDNS,
			Recursive:       recursive,
			Permutation:     permute,
			WildcardFilter:  wildcardFilter,
			OutputFormat:    getOutputFormat(),
			OutputFile:      outputFile,
			AllSources:      allSources,
			OnlyResolved:    onlyResolved,
			RemoveDuplicate: removeDuplicate,
			Silent:          !verbose,
			ShowStats:       showStats,
			ConfigPath:      configPath,
			Wordlist:        wordlistPath,
		}

		if sourcesList != "" {
			cfg.IncludeSources = strings.Split(sourcesList, ",")
			for i := range cfg.IncludeSources {
				cfg.IncludeSources[i] = strings.TrimSpace(cfg.IncludeSources[i])
			}
		}
		if excludeSourcesList != "" {
			cfg.ExcludeSources = strings.Split(excludeSourcesList, ",")
			for i := range cfg.ExcludeSources {
				cfg.ExcludeSources[i] = strings.TrimSpace(cfg.ExcludeSources[i])
			}
		}

		// Load provider config for API keys
		providerCfg, err := config.LoadProviderConfig(cfg.ConfigPath)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "[!] Warning: could not load provider config: %v\n", err)
			}
			providerCfg = config.DefaultProviderConfig()
		}
		cfg.ProviderConfig = providerCfg

		// Gather domains
		domains := []string{}
		if domain != "" {
			domains = append(domains, domain)
		}
		if domainList != "" {
			fileDomains, err := readLines(domainList)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading domain list: %v\n", err)
				os.Exit(1)
			}
			domains = append(domains, fileDomains...)
		}

		// Create and run
		r, err := runner.NewRunner(cfg, domains)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing runner: %v\n", err)
			os.Exit(1)
		}

		if err := r.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&domain, "domain", "d", "", "Target domain to enumerate")
	rootCmd.PersistentFlags().StringVarP(&domainList, "domain-list", "l", "", "File containing list of domains")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Write output to file")
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&csvOutput, "csv", "c", false, "Output in CSV format")
	rootCmd.PersistentFlags().IntVarP(&threads, "threads", "t", 10, "Number of concurrent threads per source")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 30, "HTTP timeout in seconds")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "Enable recursive subdomain enumeration")
	rootCmd.PersistentFlags().IntVar(&depth, "depth", 2, "Maximum recursion depth")
	rootCmd.PersistentFlags().IntVar(&rateLimit, "rate-limit", 0, "Max requests per second per source (0=unlimited)")
	rootCmd.PersistentFlags().BoolVar(&resolveDNS, "resolve", false, "Resolve discovered subdomains via DNS")
	rootCmd.PersistentFlags().BoolVar(&wildcardFilter, "wildcard", false, "Filter wildcard subdomains")
	rootCmd.PersistentFlags().BoolVar(&permute, "permute", false, "Enable subdomain permutation")
	rootCmd.PersistentFlags().StringVar(&wordlistPath, "wordlist", "", "Custom wordlist for permutation")
	rootCmd.PersistentFlags().StringVar(&sourcesList, "sources", "", "Comma-separated list of specific sources to use")
	rootCmd.PersistentFlags().StringVar(&excludeSourcesList, "exclude-sources", "", "Comma-separated list of sources to exclude")
	rootCmd.PersistentFlags().BoolVar(&allSources, "all-sources", false, "Use all sources including API-based ones")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose mode (show banner, progress, and statistics)")
	rootCmd.PersistentFlags().BoolVar(&showStats, "stats", true, "Show statistics after enumeration")
	rootCmd.PersistentFlags().BoolVar(&showSources, "list-sources", false, "List all available sources and exit")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to provider config file")
	rootCmd.PersistentFlags().BoolVar(&onlyResolved, "only-resolved", false, "Only show resolved subdomains")
	rootCmd.PersistentFlags().BoolVar(&removeDuplicate, "unique", false, "Remove duplicate subdomains (default: true)")
	rootCmd.SetHelpTemplate(fmt.Sprintf("Subix %s\n", version))
	rootCmd.Flags().BoolP("help", "h", false, "Help for Subix")
	removeDuplicate = true // default on
	rootCmd.SetHelpFunc(customHelp)
}

func getOutputFormat() string {
	if jsonOutput {
		return "json"
	}
	if csvOutput {
		return "csv"
	}
	return "txt"
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			result = append(result, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func printAvailableSources() {
	categories := map[string][]string{
		"Certificate Transparency": {"crtsh", "censys", "certspotter"},
		"DNS / Services":           {"dnsdumpster", "hackertarget", "urlscan", "alienvault", "anubis", "subdomaincenter", "threatcrowd", "columbus", "jldc", "sonar", "robtex", "rapiddns", "synapsint", "riddler"},
		"Web Archives":             {"wayback", "commoncrawl"},
		"Search Engines":           {"bing", "duckduckgo", "google", "yahoo", "baidu", "yandex", "ask"},
		"API (requires keys)":      {"securitytrails", "virustotal", "shodan", "passivetotal", "chaos", "bevigil", "zoomeye", "fofa", "hunter", "intelx", "leakix", "netlas", "binaryedge", "threatbook", "quake", "c99"},
	}
	colors := []string{"\033[36m", "\033[33m", "\033[35m", "\033[32m", "\033[31m"}
	i := 0
	fmt.Println("\n available sources:")
	for cat, srcs := range categories {
		fmt.Printf("  %s%s:\033[0m\n", colors[i%len(colors)], cat)
		for _, s := range srcs {
			fmt.Printf("    - %s\n", s)
		}
		i++
	}
	fmt.Println()
}

func customHelp(cmd *cobra.Command, args []string) {
	banner := `
    ███████╗██╗   ██╗██████╗ ██╗██╗  ██╗
    ██╔════╝██║   ██║██╔══██╗██║╚██╗██╔╝
    ███████╗██║   ██║██████╔╝██║ ╚███╔╝ 
    ╚════██║██║   ██║██╔══██╗██║ ██╔██╗ 
    ███████║╚██████╔╝██████╔╝██║██╔╝ ██╗
    ╚══════╝ ╚═════╝ ╚═════╝ ╚═╝╚═╝  ╚═╝
`
	fmt.Printf("\033[36m%s\033[0m", banner)
	fmt.Println("  \033[1m\033[33mSUBIX v2.0.0\033[0m | High-Impact Subdomain Enumeration Engine")
	fmt.Println("  By whoami_404 | https://github.com/youwannahackme/subix")
	fmt.Println()

	fmt.Println("\033[1mUSAGE:\033[0m")
	fmt.Println("  subix [flags]")
	fmt.Println()

	fmt.Println("\033[1mEXAMPLES:\033[0m")
	fmt.Println("  subix -d example.com                          Passive enumeration (free sources)")
	fmt.Println("  subix -d example.com -resolve -wildcard       Active enumeration with wildcard filtering")
	fmt.Println("  subix -d example.com -recursive -depth 2      Recursive passive scanning")
	fmt.Println("  subix -d example.com -permute                 Run permutation engine on found subdomains")
	fmt.Println("  subix -l domains.txt -t 50 -o results.json -j Multi-domain scan with JSON output")
	fmt.Println()

	fmt.Println("\033[1mTARGETS:\033[0m")
	fmt.Println("  -d, --domain <string>        Target domain to enumerate")
	fmt.Println("  -l, --domain-list <path>     File containing list of domains to enumerate")
	fmt.Println()

	fmt.Println("\033[1mCONFIGURATION:\033[0m")
	fmt.Println("      --config <path>          Path to provider config file")
	fmt.Println("      --sources <list>         Comma-separated list of passive sources to use")
	fmt.Println("      --exclude-sources <list> Comma-separated list of passive sources to exclude")
	fmt.Println("      --all-sources            Use all sources including API-based ones")
	fmt.Println()

	fmt.Println("\033[1mOPTIMIZATION:\033[0m")
	fmt.Println("  -t, --threads <int>          Number of concurrent threads (default: 10)")
	fmt.Println("      --timeout <int>          HTTP timeout in seconds (default: 30)")
	fmt.Println("      --rate-limit <int>       Max requests per second per source (default: unlimited)")
	fmt.Println()

	fmt.Println("\033[1mADVANCED:\033[0m")
	fmt.Println("  -r, --recursive              Enable recursive subdomain enumeration")
	fmt.Println("      --depth <int>            Maximum recursion depth (default: 2)")
	fmt.Println("      --permute                Enable subdomain permutation")
	fmt.Println("      --wordlist <path>        Custom wordlist for permutation")
	fmt.Println("      --resolve                Resolve discovered subdomains via DNS")
	fmt.Println("      --only-resolved          Only show resolved subdomains in output")
	fmt.Println("      --wildcard               Filter wildcard subdomains")
	fmt.Println("      --unique                 Remove duplicate subdomains (default: true)")
	fmt.Println()

	fmt.Println("\033[1mOUTPUT:\033[0m")
	fmt.Println("  -o, --output <path>          Write output to file")
	fmt.Println("  -j, --json                   Output in JSON format")
	fmt.Println("  -c, --csv                    Output in CSV format")
	fmt.Println("  -v, --verbose                Verbose mode (show progress and stats)")
	fmt.Println("      --stats                  Show statistics after enumeration (default: true)")
	fmt.Println()

	fmt.Println("\033[1mSYSTEM:\033[0m")
	fmt.Println("  -h, --help                   Show this help menu")
	fmt.Println("      --list-sources           List all 42 available sources and exit")
	fmt.Println()
}
