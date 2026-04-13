package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/exporters/obsidian"
	"github.com/spf13/cobra"
)

var exportObsidianCmd = &cobra.Command{
	Use:   "obsidian",
	Short: "Export graph as Obsidian vault",
	Long: `Export the graph as an Obsidian vault with wikilinks.

Creates a wiki-style structure with:
  - index.md: Entry point with god nodes and overview
  - communities/: One page per detected community
  - nodes/: One page per significant node (functions, types)

Pages are interconnected with [[wikilinks]] for easy navigation.

Examples:
  graphize export obsidian -o ./vault
  graphize export obsidian -o ~/obsidian/code-graph`,
	RunE: runExportObsidian,
}

var (
	obsidianOutput string
	obsidianTopN   int
	obsidianMinDeg int
)

func init() {
	exportCmd.AddCommand(exportObsidianCmd)
	exportObsidianCmd.Flags().StringVarP(&obsidianOutput, "output", "o", "", "Output directory (required)")
	exportObsidianCmd.Flags().IntVar(&obsidianTopN, "top", 20, "Number of top nodes to include as individual pages")
	exportObsidianCmd.Flags().IntVar(&obsidianMinDeg, "min-degree", 3, "Minimum degree for a node to get its own page")
	_ = exportObsidianCmd.MarkFlagRequired("output")
}

func runExportObsidian(cmd *cobra.Command, args []string) error {
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

	// Create output directories
	if err := os.MkdirAll(obsidianOutput, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(obsidianOutput, "communities"), 0755); err != nil {
		return fmt.Errorf("creating communities directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(obsidianOutput, "nodes"), 0755); err != nil {
		return fmt.Errorf("creating nodes directory: %w", err)
	}

	// Generate vault content using the library
	gen := &obsidian.Generator{
		TopN:      obsidianTopN,
		MinDegree: obsidianMinDeg,
	}
	vault := gen.Generate(nodes, edges)

	// Write index.md
	if err := os.WriteFile(filepath.Join(obsidianOutput, "index.md"), []byte(vault.Index), 0600); err != nil {
		return fmt.Errorf("writing index: %w", err)
	}

	// Write community pages
	for commID, content := range vault.Communities {
		filename := fmt.Sprintf("community-%d.md", commID)
		if err := os.WriteFile(filepath.Join(obsidianOutput, "communities", filename), []byte(content), 0600); err != nil {
			return fmt.Errorf("writing community %d: %w", commID, err)
		}
	}

	// Write node pages
	for nodeName, content := range vault.Nodes {
		filename := nodeName + ".md"
		if err := os.WriteFile(filepath.Join(obsidianOutput, "nodes", filename), []byte(content), 0600); err != nil {
			return fmt.Errorf("writing node %s: %w", nodeName, err)
		}
	}

	// Report
	fmt.Printf("Exported Obsidian vault to %s\n", obsidianOutput)
	fmt.Printf("  index.md: Overview\n")
	fmt.Printf("  communities/: %d community pages\n", len(vault.Communities))
	fmt.Printf("  nodes/: %d node pages\n", len(vault.Nodes))

	return nil
}
