package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/analyze"
	"github.com/spf13/cobra"
)

var diffOldGraph string

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare two graph snapshots",
	Long: `Compare two graph snapshots and show what changed.

Shows:
  - New nodes added
  - Nodes removed
  - New edges added
  - Edges removed

Examples:
  graphize diff --old /path/to/old/graph --graph /path/to/new/graph
  graphize diff --old .graphize.bak/graph -g .graphize/graph`,
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().StringVar(&diffOldGraph, "old", "", "Path to the old graph (required)")
	diffCmd.MarkFlagRequired("old")
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Load old graph
	oldPath, err := filepath.Abs(diffOldGraph)
	if err != nil {
		return fmt.Errorf("resolving old graph path: %w", err)
	}

	oldStore, err := store.NewFSStore(oldPath)
	if err != nil {
		return fmt.Errorf("opening old graph: %w", err)
	}

	oldNodes, err := oldStore.ListNodes()
	if err != nil {
		return fmt.Errorf("listing old nodes: %w", err)
	}

	oldEdges, err := oldStore.ListEdges()
	if err != nil {
		return fmt.Errorf("listing old edges: %w", err)
	}

	// Load new graph
	newPath, err := filepath.Abs(graphPath)
	if err != nil {
		return fmt.Errorf("resolving new graph path: %w", err)
	}

	newStore, err := store.NewFSStore(newPath)
	if err != nil {
		return fmt.Errorf("opening new graph: %w", err)
	}

	newNodes, err := newStore.ListNodes()
	if err != nil {
		return fmt.Errorf("listing new nodes: %w", err)
	}

	newEdges, err := newStore.ListEdges()
	if err != nil {
		return fmt.Errorf("listing new edges: %w", err)
	}

	// Compute diff
	diff := analyze.DiffGraphs(oldNodes, newNodes, oldEdges, newEdges)

	// Output based on format
	switch outputFormat {
	case "json":
		data, err := json.MarshalIndent(diff, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling diff: %w", err)
		}
		fmt.Println(string(data))
	default:
		// Human-readable output
		fmt.Printf("Graph Diff: %s → %s\n\n", oldPath, newPath)
		fmt.Printf("Summary: %s\n\n", diff.Summary)

		if len(diff.NewNodes) > 0 {
			fmt.Printf("New Nodes (%d):\n", len(diff.NewNodes))
			for _, n := range diff.NewNodes {
				label := n.Label
				if label == "" {
					label = n.ID
				}
				fmt.Printf("  + %s (%s)\n", label, n.Type)
			}
			fmt.Println()
		}

		if len(diff.RemovedNodes) > 0 {
			fmt.Printf("Removed Nodes (%d):\n", len(diff.RemovedNodes))
			for _, n := range diff.RemovedNodes {
				label := n.Label
				if label == "" {
					label = n.ID
				}
				fmt.Printf("  - %s (%s)\n", label, n.Type)
			}
			fmt.Println()
		}

		if len(diff.NewEdges) > 0 {
			fmt.Printf("New Edges (%d):\n", len(diff.NewEdges))
			shown := diff.NewEdges
			if len(shown) > 10 {
				shown = shown[:10]
			}
			for _, e := range shown {
				fmt.Printf("  + %s --%s--> %s\n", e.From, e.Type, e.To)
			}
			if len(diff.NewEdges) > 10 {
				fmt.Printf("  ... and %d more\n", len(diff.NewEdges)-10)
			}
			fmt.Println()
		}

		if len(diff.RemovedEdges) > 0 {
			fmt.Printf("Removed Edges (%d):\n", len(diff.RemovedEdges))
			shown := diff.RemovedEdges
			if len(shown) > 10 {
				shown = shown[:10]
			}
			for _, e := range shown {
				fmt.Printf("  - %s --%s--> %s\n", e.From, e.Type, e.To)
			}
			if len(diff.RemovedEdges) > 10 {
				fmt.Printf("  ... and %d more\n", len(diff.RemovedEdges)-10)
			}
		}
	}

	return nil
}
