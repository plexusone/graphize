package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/reuse"
	"github.com/spf13/cobra"
)

var reuseCmd = &cobra.Command{
	Use:   "reuse",
	Short: "Analyze code reuse patterns and similarity",
	Long: `Analyze the knowledge graph for code reuse opportunities.

Identifies:
  - Similar function signatures that could share an interface
  - Duplicate names across packages
  - Shared dependencies suggesting common abstractions
  - Refactoring candidates

Examples:
  graphize reuse                    # Run full reuse analysis
  graphize reuse --format json      # Output as JSON
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve graph path
		absGraphPath, err := filepath.Abs(graphPath)
		if err != nil {
			return fmt.Errorf("resolving graph path: %w", err)
		}

		// Load graph
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
			return fmt.Errorf("no nodes found in graph at %s", absGraphPath)
		}

		// Run analysis
		tracker := reuse.NewTracker(nodes, edges)
		report := tracker.Analyze()

		return printOutput(report)
	},
}

func init() {
	rootCmd.AddCommand(reuseCmd)
}
