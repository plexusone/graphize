package cmd

import (
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
		data, err := loadGraph()
		if err != nil {
			return err
		}

		detector := patterns.NewDetector(data.Nodes, data.Edges)
		report := detector.Detect()

		return printOutput(report)
	},
}

func init() {
	rootCmd.AddCommand(patternsCmd)
}
