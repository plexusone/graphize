package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphfs/pkg/query"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/spf13/cobra"
)

var (
	pathEdgeTypes string
	pathJSON      bool
)

var pathCmd = &cobra.Command{
	Use:   "path <from> <to>",
	Short: "Find shortest path between two nodes",
	Long: `Find the shortest path between two nodes in the knowledge graph.

Uses BFS to find the path with fewest hops. Shows all intermediate
nodes and the edge types connecting them.

Examples:
  graphize path func_main func_helper
  graphize path pkg_cmd pkg_utils
  graphize path func_main func_helper --edge-type calls
  graphize path func_main func_helper --json`,
	Args: cobra.ExactArgs(2),
	RunE: runPathCmd,
}

func init() {
	rootCmd.AddCommand(pathCmd)
	pathCmd.Flags().StringVar(&pathEdgeTypes, "edge-type", "", "Filter by edge type(s), comma-separated")
	pathCmd.Flags().BoolVar(&pathJSON, "json", false, "Output as JSON")
}

func runPathCmd(cmd *cobra.Command, args []string) error {
	from := args[0]
	to := args[1]

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

	g, err := graphStore.LoadGraph()
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	// Check if nodes exist
	fromNode := g.GetNode(from)
	toNode := g.GetNode(to)

	if fromNode == nil {
		matches := findPartialMatches(g, from)
		if len(matches) > 0 {
			return fmt.Errorf("node %q not found. Did you mean: %s", from, strings.Join(matches, ", "))
		}
		return fmt.Errorf("node %q not found", from)
	}

	if toNode == nil {
		matches := findPartialMatches(g, to)
		if len(matches) > 0 {
			return fmt.Errorf("node %q not found. Did you mean: %s", to, strings.Join(matches, ", "))
		}
		return fmt.Errorf("node %q not found", to)
	}

	// Parse edge types filter
	var edgeTypes []string
	if pathEdgeTypes != "" {
		edgeTypes = strings.Split(pathEdgeTypes, ",")
	}

	// Find path
	traverser := query.NewTraverser(g)
	result := traverser.FindPath(from, to, edgeTypes)

	if len(result.Visited) == 0 {
		if pathJSON {
			output := map[string]any{
				"from":    from,
				"to":      to,
				"found":   false,
				"message": "No path found between nodes",
			}
			return printOutput(output)
		}
		fmt.Printf("No path found from %s to %s\n", from, to)
		return nil
	}

	if pathJSON {
		// JSON output
		var edgesOut []map[string]any
		for _, e := range result.Edges {
			edgesOut = append(edgesOut, map[string]any{
				"from":       e.From,
				"to":         e.To,
				"type":       e.Type,
				"confidence": e.Confidence,
			})
		}

		output := map[string]any{
			"from":   from,
			"to":     to,
			"found":  true,
			"length": len(result.Visited) - 1,
			"path":   result.Visited,
			"edges":  edgesOut,
		}
		return printOutput(output)
	}

	// Human-readable output
	fmt.Printf("Path found (%d hops):\n\n", len(result.Visited)-1)

	// Build node label map
	nodeLabels := make(map[string]string)
	for _, id := range result.Visited {
		if n := g.GetNode(id); n != nil && n.Label != "" && n.Label != id {
			nodeLabels[id] = n.Label
		}
	}

	// Print path with edges
	for i, nodeID := range result.Visited {
		label := nodeID
		if l, ok := nodeLabels[nodeID]; ok {
			label = fmt.Sprintf("%s (%s)", nodeID, l)
		}

		if i == 0 {
			fmt.Printf("  %s\n", label)
		} else {
			edge := result.Edges[i-1]
			edgeLabel := formatEdge(edge)
			fmt.Printf("    │\n")
			fmt.Printf("    ├── %s\n", edgeLabel)
			fmt.Printf("    │\n")
			fmt.Printf("    ▼\n")
			fmt.Printf("  %s\n", label)
		}
	}

	return nil
}

func findPartialMatches(g *graph.Graph, partial string) []string {
	var matches []string
	for id := range g.Nodes {
		if strings.Contains(strings.ToLower(id), strings.ToLower(partial)) {
			matches = append(matches, id)
			if len(matches) >= 5 {
				break
			}
		}
	}
	return matches
}

func formatEdge(e *graph.Edge) string {
	if e.Confidence != "" && e.Confidence != graph.ConfidenceExtracted {
		return fmt.Sprintf("[%s] (%s %.2f)", e.Type, e.Confidence, e.ConfidenceScore)
	}
	return fmt.Sprintf("[%s]", e.Type)
}
