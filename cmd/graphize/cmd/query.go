package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	gquery "github.com/plexusone/graphfs/pkg/query"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/query"
	"github.com/spf13/cobra"
)

var (
	queryFrom     string
	queryType     string
	queryLimit    int
	queryDepth    int
	queryDFS      bool
	queryDir      string
	queryPath     string
	queryEdgeType string
)

var queryCmd = &cobra.Command{
	Use:   "query [node-id]",
	Short: "Query the knowledge graph",
	Long: `Query the extracted knowledge graph.

Examples:
  graphize query                              # Show graph summary
  graphize query func_main                    # Show edges for a node
  graphize query func_main --depth 3          # BFS traverse 3 levels deep
  graphize query func_main --dfs --depth 5    # DFS traverse
  graphize query func_main --dir out          # Only outgoing edges (what does X call?)
  graphize query func_main --dir in           # Only incoming edges (what calls X?)
  graphize query --path nodeA nodeB           # Find path between nodes
  graphize query --type calls                 # Filter by edge type
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

		// Path finding mode
		if queryPath != "" && len(args) > 0 {
			return findPath(graphStore, queryPath, args[0])
		}

		// If no args and no filters, show summary
		if len(args) == 0 && queryFrom == "" && queryType == "" {
			return showSummary(graphStore)
		}

		// Traversal mode (BFS/DFS)
		if len(args) > 0 && queryDepth > 0 {
			return traverseGraph(graphStore, args[0])
		}

		// Simple edge listing
		return listEdges(graphStore, args)
	},
}

func traverseGraph(graphStore *store.FSStore, startNode string) error {
	// Load full graph
	g, err := graphStore.LoadGraph()
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	// Check if start node exists
	if g.GetNode(startNode) == nil {
		matches := query.FindPartialMatches(g, startNode, 10)
		if len(matches.Matches) == 0 {
			return fmt.Errorf("node %q not found", startNode)
		}
		result := map[string]any{
			"error":   "node not found",
			"query":   startNode,
			"matches": matches.Matches,
			"message": matches.Message,
		}
		return printOutput(result)
	}

	// Create traverser
	traverser := gquery.NewTraverser(g)

	// Determine direction
	dir := gquery.Both
	switch queryDir {
	case "out", "outgoing":
		dir = gquery.Outgoing
	case "in", "incoming":
		dir = gquery.Incoming
	}

	// Parse edge types filter
	var edgeTypes []string
	if queryEdgeType != "" {
		edgeTypes = strings.Split(queryEdgeType, ",")
	}

	// Perform traversal
	var result *gquery.TraversalResult
	if queryDFS {
		result = traverser.DFS(startNode, dir, queryDepth, edgeTypes)
	} else {
		result = traverser.BFS(startNode, dir, queryDepth, edgeTypes)
	}

	// Format output using pkg/query
	algorithm := "BFS"
	if queryDFS {
		algorithm = "DFS"
	}

	output := query.FormatTraversal(result, startNode, queryDepth, query.FormatTraversalOptions{
		Limit:     queryLimit,
		Algorithm: algorithm,
		Direction: queryDir,
	})

	return printOutput(output)
}

func findPath(graphStore *store.FSStore, from, to string) error {
	// Load full graph
	g, err := graphStore.LoadGraph()
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	// Create traverser
	traverser := gquery.NewTraverser(g)

	// Parse edge types filter
	var edgeTypes []string
	if queryEdgeType != "" {
		edgeTypes = strings.Split(queryEdgeType, ",")
	}

	// Find path
	result := traverser.FindPath(from, to, edgeTypes)

	// Format output using pkg/query
	output := query.FormatPath(result, from, to)

	return printOutput(output)
}

func listEdges(graphStore *store.FSStore, args []string) error {
	// Load edges
	edges, err := graphStore.ListEdges()
	if err != nil {
		return fmt.Errorf("loading edges: %w", err)
	}

	// Build filter
	filter := query.EdgeFilter{
		From:  queryFrom,
		Type:  queryType,
		Limit: queryLimit,
	}
	if len(args) > 0 {
		filter.NodeID = args[0]
	}

	// Filter and format using pkg/query
	output := query.FilterEdges(edges, filter)

	return printOutput(output)
}

func showSummary(graphStore *store.FSStore) error {
	nodes, err := graphStore.ListNodes()
	if err != nil {
		return fmt.Errorf("loading nodes: %w", err)
	}

	edges, err := graphStore.ListEdges()
	if err != nil {
		return fmt.Errorf("loading edges: %w", err)
	}

	// Compute summary using pkg/query
	summary := query.ComputeSummaryFromLists(nodes, edges)

	result := map[string]any{
		"total_nodes": summary.TotalNodes,
		"total_edges": summary.TotalEdges,
		"node_types":  summary.NodeTypes,
		"edge_types":  summary.EdgeTypes,
		"god_nodes":   summary.GodNodes,
		"message":     "Use 'graphize query <node-id> --depth N' to traverse from a node.",
	}

	return printOutput(result)
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().StringVar(&queryFrom, "from", "", "Filter edges by source node")
	queryCmd.Flags().StringVar(&queryType, "type", "", "Filter edges by type (calls, imports, contains, etc)")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 100, "Maximum number of results")
	queryCmd.Flags().IntVar(&queryDepth, "depth", 0, "Traversal depth (enables BFS/DFS mode)")
	queryCmd.Flags().BoolVar(&queryDFS, "dfs", false, "Use depth-first search instead of breadth-first")
	queryCmd.Flags().StringVar(&queryDir, "dir", "both", "Traversal direction: out (outgoing), in (incoming), both")
	queryCmd.Flags().StringVar(&queryPath, "path", "", "Find path from this node to the argument node")
	queryCmd.Flags().StringVar(&queryEdgeType, "edge-type", "", "Filter traversal by edge type(s), comma-separated")
}
