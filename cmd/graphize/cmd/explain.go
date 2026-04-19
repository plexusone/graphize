package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/query"
	"github.com/spf13/cobra"
)

var (
	explainDepth    int
	explainJSON     bool
	explainMaxEdges int
)

var explainCmd = &cobra.Command{
	Use:   "explain <node-id>",
	Short: "Explain a node in context",
	Long: `Provide comprehensive context about a specific node in the graph.

Shows:
  - Node metadata (type, label, source file)
  - Connectivity (incoming/outgoing edges, degree)
  - Community membership and bridge status
  - Centrality metrics (betweenness rank)

Examples:
  graphize explain func_main
  graphize explain type_Server --json
  graphize explain method_Handler.ServeHTTP --depth 2
`,
	Args: cobra.ExactArgs(1),
	RunE: runExplain,
}

func init() {
	rootCmd.AddCommand(explainCmd)
	explainCmd.Flags().IntVar(&explainDepth, "depth", 1, "Neighbor depth to include (1-3)")
	explainCmd.Flags().BoolVar(&explainJSON, "json", false, "Output in JSON format")
	explainCmd.Flags().IntVar(&explainMaxEdges, "max-edges", 20, "Maximum edges to show per direction")
}

func runExplain(cmd *cobra.Command, args []string) error {
	nodeID := args[0]

	// Resolve graph path
	absGraphPath, err := filepath.Abs(graphPath)
	if err != nil {
		return fmt.Errorf("resolving graph path: %w", err)
	}

	// Load graph
	s, err := store.NewFSStore(absGraphPath)
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

	// Build graph for partial matching
	g, err := s.LoadGraph()
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	// Check if node exists, suggest alternatives if not
	if g.GetNode(nodeID) == nil {
		matches := query.FindPartialMatches(g, nodeID, 10)
		if len(matches.Matches) == 0 {
			return fmt.Errorf("node %q not found", nodeID)
		}

		if explainJSON {
			data, _ := json.MarshalIndent(map[string]any{
				"error":   "node not found",
				"query":   nodeID,
				"matches": matches.Matches,
				"message": matches.Message,
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Node %q not found.\n\n%s\n", nodeID, matches.Message)
		for _, m := range matches.Matches {
			fmt.Printf("  - %s\n", m)
		}
		return nil
	}

	// Get explanation
	opts := query.ExplainOptions{
		Depth:        explainDepth,
		MaxNeighbors: explainMaxEdges,
		IncludeEdges: true,
	}

	explanation, err := query.ExplainNode(nodes, edges, nodeID, opts)
	if err != nil {
		return fmt.Errorf("explaining node: %w", err)
	}

	// Output
	if explainJSON {
		data, err := json.MarshalIndent(explanation, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Print(query.FormatExplanation(explanation))
	}

	return nil
}
