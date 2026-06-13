package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/patterns"
	"github.com/spf13/cobra"
)

var patternsCmd = &cobra.Command{
	Use:   "patterns",
	Short: "Detect architectural and structural patterns",
	Long: `Detect common patterns in the knowledge graph.

Architectural patterns detected:
  - Factory: New* functions that create instances
  - Singleton: Global instances used widely
  - Handler: HTTP/RPC request handlers
  - Repository: Data access layer components
  - Builder: Fluent builder patterns

Structural patterns detected:
  - Hub nodes: Highly connected central components
  - Layered architecture: Presentation/business/data layers
  - Clusters: Tightly coupled node groups

Anti-patterns detected:
  - God objects: Components with too many responsibilities
  - Circular dependencies: Mutual dependencies between nodes
  - Dead code: Potentially unused functions

Examples:
  graphize patterns                 # Run full pattern detection
  graphize patterns --format json   # Output as JSON
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

		// Run detection
		detector := patterns.NewDetector(nodes, edges)
		report := detector.Detect()

		return printOutput(report)
	},
}

func init() {
	rootCmd.AddCommand(patternsCmd)
}
