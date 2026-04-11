package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/output"
	"github.com/spf13/cobra"
)

var exportToonCmd = &cobra.Command{
	Use:   "toon",
	Short: "Export graph in TOON format (agent-optimized)",
	Long: `Export the graph in TOON format, optimized for AI agent consumption.

TOON format is token-efficient and easy to parse. The output includes:
  - All nodes grouped by type
  - All edges grouped by type
  - Minimal metadata (no absolute paths)

Use --gzip to compress the output (recommended for large graphs).

Examples:
  graphize export toon -o AGENTS/GRAPH.toon
  graphize export toon -o AGENTS/GRAPH.toon.gz --gzip
  graphize export toon --no-extra  # Exclude source locations`,
	RunE: runExportToon,
}

var (
	toonOutput  string
	toonGzip    bool
	toonNoExtra bool
	toonCompact bool
)

func init() {
	exportCmd.AddCommand(exportToonCmd)
	exportToonCmd.Flags().StringVarP(&toonOutput, "output", "o", "", "Output file path")
	exportToonCmd.Flags().BoolVar(&toonGzip, "gzip", false, "Gzip compress the output")
	exportToonCmd.Flags().BoolVar(&toonNoExtra, "no-extra", false, "Exclude extra metadata (source locations)")
	exportToonCmd.Flags().BoolVar(&toonCompact, "compact", false, "Compact format (shorter IDs, no extra)")
}

func runExportToon(cmd *cobra.Command, args []string) error {
	path := graphPath
	if path == "" {
		path = ".graphize"
	}

	s, err := store.NewFSStore(path)
	if err != nil {
		return fmt.Errorf("opening graph store: %w", err)
	}

	nodes, err := s.ListNodes()
	if err != nil {
		return fmt.Errorf("listing nodes: %w", err)
	}

	edges, err := s.ListEdges()
	if err != nil {
		return fmt.Errorf("listing edges: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found. Run 'graphize analyze' first")
	}

	// Generate TOON content
	content := output.GenerateTOON(nodes, edges, output.TOONOptions{
		NoExtra: toonNoExtra,
		Compact: toonCompact,
	})

	// Determine output
	outputPath := toonOutput
	if outputPath == "" {
		if toonGzip {
			outputPath = "graph.toon.gz"
		} else {
			outputPath = "graph.toon"
		}
	}

	// Ensure directory exists
	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	// Write output
	if toonGzip || strings.HasSuffix(outputPath, ".gz") {
		if err := output.WriteTOONGzipped(outputPath, content); err != nil {
			return err
		}
	} else {
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing TOON file: %w", err)
		}
	}

	// Report stats
	fi, _ := os.Stat(outputPath)
	fmt.Printf("Exported graph to %s\n", outputPath)
	fmt.Printf("  Nodes: %d\n", len(nodes))
	fmt.Printf("  Edges: %d\n", len(edges))
	if fi != nil {
		fmt.Printf("  Size: %s\n", output.FormatSize(fi.Size()))
	}

	return nil
}
