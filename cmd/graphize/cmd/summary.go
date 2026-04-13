package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/output"
	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generate a markdown summary of the graph",
	Long: `Generate a lightweight markdown summary suitable for agents.

The summary includes:
  - Node and edge statistics by type
  - Top nodes by connection count (hubs)
  - Package/module structure overview

This is designed to be small enough to check into version control
as context for AI agents, without the overhead of the full graph.

Examples:
  graphize summary -o GRAPH_SUMMARY.md
  graphize summary --top 20`,
	RunE: runSummary,
}

var (
	summaryOutput string
	summaryTop    int
)

func init() {
	rootCmd.AddCommand(summaryCmd)
	summaryCmd.Flags().StringVarP(&summaryOutput, "output", "o", "", "Output file (default: stdout)")
	summaryCmd.Flags().IntVar(&summaryTop, "top", 10, "Number of top nodes to show per category")
}

func runSummary(cmd *cobra.Command, args []string) error {
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

	// Generate markdown summary
	markdown := output.GenerateSummaryMarkdown(nodes, edges, output.SummaryOptions{
		TopN: summaryTop,
	})

	if summaryOutput != "" {
		if dir := filepath.Dir(summaryOutput); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating output directory: %w", err)
			}
		}
		if err := os.WriteFile(summaryOutput, []byte(markdown), 0600); err != nil {
			return fmt.Errorf("writing summary: %w", err)
		}
		fmt.Printf("Generated summary: %s\n", summaryOutput)
	} else {
		fmt.Print(markdown)
	}

	return nil
}
