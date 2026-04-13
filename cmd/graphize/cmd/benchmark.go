package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/output"
	"github.com/plexusone/graphize/pkg/source"
	"github.com/spf13/cobra"
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Show token reduction statistics",
	Long: `Measure token reduction between raw source files and TOON export.

Compares the token count of raw source files versus the compact TOON
format, showing how much context is saved when using graphize output
instead of raw files.

Token estimation uses ~4 characters per token (GPT tokenizer average).

Examples:
  graphize benchmark
  graphize benchmark --json`,
	RunE: runBenchmark,
}

var benchmarkJSON bool

func init() {
	rootCmd.AddCommand(benchmarkCmd)
	benchmarkCmd.Flags().BoolVar(&benchmarkJSON, "json", false, "Output as JSON")
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	// Resolve paths
	absGraphPath, err := filepath.Abs(graphPath)
	if err != nil {
		return fmt.Errorf("resolving graph path: %w", err)
	}

	// Load manifest to get source files
	manifestPath := filepath.Join(absGraphPath, "manifest.json")
	manifest, err := source.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if len(manifest.Sources) == 0 {
		return fmt.Errorf("no sources tracked. Run 'graphize add <path>' first")
	}

	// Count tokens in source files
	var totalSourceTokens int
	var totalSourceBytes int
	var fileCount int

	for _, src := range manifest.Sources {
		srcPath := src.Path
		if !filepath.IsAbs(srcPath) {
			srcPath = filepath.Join(filepath.Dir(absGraphPath), srcPath)
		}

		tokens, bytes, files, err := countSourceTokens(srcPath)
		if err != nil {
			// Skip sources that can't be read
			continue
		}
		totalSourceTokens += tokens
		totalSourceBytes += bytes
		fileCount += files
	}

	if totalSourceTokens == 0 {
		return fmt.Errorf("no source files found to analyze")
	}

	// Load graph and generate TOON
	graphStore, err := store.NewFSStore(absGraphPath)
	if err != nil {
		return fmt.Errorf("opening graph store: %w", err)
	}

	nodes, err := graphStore.ListNodes()
	if err != nil {
		return fmt.Errorf("loading nodes: %w", err)
	}

	edges, err := graphStore.ListEdges()
	if err != nil {
		return fmt.Errorf("loading edges: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes in graph. Run 'graphize analyze' first")
	}

	// Generate TOON output
	toonOutput := output.GenerateTOON(nodes, edges, output.TOONOptions{
		Compact: true,
		NoExtra: true,
	})
	toonTokens := estimateTokens(toonOutput)
	toonBytes := len(toonOutput)

	// Calculate reduction
	var reduction float64
	if toonTokens > 0 {
		reduction = float64(totalSourceTokens) / float64(toonTokens)
	}

	if benchmarkJSON {
		result := map[string]any{
			"source": map[string]any{
				"files":  fileCount,
				"bytes":  totalSourceBytes,
				"tokens": totalSourceTokens,
			},
			"toon": map[string]any{
				"bytes":  toonBytes,
				"tokens": toonTokens,
			},
			"graph": map[string]any{
				"nodes": len(nodes),
				"edges": len(edges),
			},
			"reduction":   fmt.Sprintf("%.1fx", reduction),
			"compression": fmt.Sprintf("%.1f%%", (1-float64(toonTokens)/float64(totalSourceTokens))*100),
		}
		return printOutput(result)
	}

	// Human-readable output
	fmt.Println("Token Reduction Benchmark")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Source Files:")
	fmt.Printf("  Files:  %d\n", fileCount)
	fmt.Printf("  Bytes:  %s\n", formatBytes(totalSourceBytes))
	fmt.Printf("  Tokens: %s\n", formatNumber(totalSourceTokens))
	fmt.Println()
	fmt.Println("TOON Export:")
	fmt.Printf("  Bytes:  %s\n", formatBytes(toonBytes))
	fmt.Printf("  Tokens: %s\n", formatNumber(toonTokens))
	fmt.Println()
	fmt.Println("Graph Statistics:")
	fmt.Printf("  Nodes:  %d\n", len(nodes))
	fmt.Printf("  Edges:  %d\n", len(edges))
	fmt.Println()
	fmt.Printf("Reduction:   %.1fx fewer tokens\n", reduction)
	fmt.Printf("Compression: %.1f%% smaller\n", (1-float64(toonTokens)/float64(totalSourceTokens))*100)

	return nil
}

// countSourceTokens recursively counts tokens in Go source files
func countSourceTokens(root string) (tokens, bytes, files int, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			// Skip hidden and vendor directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only count Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files for cleaner comparison
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path) //nolint:gosec // G122: path is from filepath.Walk, not user input
		if err != nil {
			return nil // Skip unreadable files
		}

		bytes += len(content)
		tokens += estimateTokens(string(content))
		files++

		return nil
	})

	return tokens, bytes, files, err
}

// estimateTokens estimates token count using ~4 characters per token
func estimateTokens(text string) int {
	// More accurate estimation: count words and multiply by average tokens per word
	// GPT models average ~1.3 tokens per word for code
	words := 0
	inWord := false

	for _, r := range text {
		isWordChar := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
		if isWordChar && !inWord {
			words++
			inWord = true
		} else if !isWordChar {
			inWord = false
		}
	}

	// Code typically has more tokens per word due to punctuation
	// Estimate: words * 1.5 + punctuation
	punctuation := 0
	for _, r := range text {
		if unicode.IsPunct(r) || r == '{' || r == '}' || r == '(' || r == ')' {
			punctuation++
		}
	}

	return int(float64(words)*1.5) + punctuation/2
}

func formatBytes(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}
