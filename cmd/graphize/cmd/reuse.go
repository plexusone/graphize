package cmd

import (
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
		data, err := loadGraph()
		if err != nil {
			return err
		}

		tracker := reuse.NewTracker(data.Nodes, data.Edges)
		report := tracker.Analyze()

		return printOutput(report)
	},
}

func init() {
	rootCmd.AddCommand(reuseCmd)
}
