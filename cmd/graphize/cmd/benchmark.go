package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/metrics"
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
	toonTokens := metrics.EstimateTokens(toonOutput)
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
	fmt.Printf("  Bytes:  %s\n", metrics.FormatBytes(int64(totalSourceBytes)))
	fmt.Printf("  Tokens: %s\n", metrics.FormatNumber(totalSourceTokens))
	fmt.Println()
	fmt.Println("TOON Export:")
	fmt.Printf("  Bytes:  %s\n", metrics.FormatBytes(int64(toonBytes)))
	fmt.Printf("  Tokens: %s\n", metrics.FormatNumber(toonTokens))
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
	opts := metrics.WalkOptions{
		Extensions: []string{".go"},
		SkipDirs:   []string{"vendor", "node_modules"},
		SkipHidden: true,
		SkipTests:  true,
	}

	err = metrics.WalkSourceFilesWithContent(root, opts, func(path string, content []byte) error {
		bytes += len(content)
		tokens += metrics.EstimateTokensInFile(content)
		files++
		return nil
	})

	return tokens, bytes, files, err
}
