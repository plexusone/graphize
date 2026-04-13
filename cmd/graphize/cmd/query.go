package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/plexusone/graphfs/pkg/query"
	"github.com/plexusone/graphfs/pkg/store"
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
		// Try to find partial match
		var matches []string
		for id := range g.Nodes {
			if strings.Contains(id, startNode) {
				matches = append(matches, id)
			}
		}
		if len(matches) == 0 {
			return fmt.Errorf("node %q not found", startNode)
		}
		if len(matches) > 10 {
			matches = matches[:10]
		}
		result := map[string]any{
			"error":   "node not found",
			"query":   startNode,
			"matches": matches,
			"message": "Did you mean one of these?",
		}
		return printOutput(result)
	}

	// Create traverser
	traverser := query.NewTraverser(g)

	// Determine direction
	dir := query.Both
	switch queryDir {
	case "out", "outgoing":
		dir = query.Outgoing
	case "in", "incoming":
		dir = query.Incoming
	}

	// Parse edge types filter
	var edgeTypes []string
	if queryEdgeType != "" {
		edgeTypes = strings.Split(queryEdgeType, ",")
	}

	// Perform traversal
	var result *query.TraversalResult
	if queryDFS {
		result = traverser.DFS(startNode, dir, queryDepth, edgeTypes)
	} else {
		result = traverser.BFS(startNode, dir, queryDepth, edgeTypes)
	}

	// Build output
	algorithm := "BFS"
	if queryDFS {
		algorithm = "DFS"
	}

	// Group nodes by depth
	nodesByDepth := make(map[int][]string)
	for node, depth := range result.Depth {
		nodesByDepth[depth] = append(nodesByDepth[depth], node)
	}

	// Build depth layers output
	var layers []map[string]any
	for d := 0; d <= queryDepth; d++ {
		nodes := nodesByDepth[d]
		if len(nodes) == 0 {
			continue
		}
		sort.Strings(nodes)
		if queryLimit > 0 && len(nodes) > queryLimit {
			nodes = nodes[:queryLimit]
		}
		layers = append(layers, map[string]any{
			"depth": d,
			"count": len(nodesByDepth[d]),
			"nodes": nodes,
		})
	}

	// Build edges output
	var edgesOut []map[string]any
	for _, e := range result.Edges {
		edgesOut = append(edgesOut, map[string]any{
			"from": e.From,
			"to":   e.To,
			"type": e.Type,
		})
	}
	if queryLimit > 0 && len(edgesOut) > queryLimit {
		edgesOut = edgesOut[:queryLimit]
	}

	output := map[string]any{
		"query":       startNode,
		"algorithm":   algorithm,
		"direction":   queryDir,
		"max_depth":   queryDepth,
		"nodes_found": len(result.Visited),
		"edges_found": len(result.Edges),
		"layers":      layers,
		"edges":       edgesOut,
	}

	return printOutput(output)
}

func findPath(graphStore *store.FSStore, from, to string) error {
	// Load full graph
	g, err := graphStore.LoadGraph()
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	// Create traverser
	traverser := query.NewTraverser(g)

	// Parse edge types filter
	var edgeTypes []string
	if queryEdgeType != "" {
		edgeTypes = strings.Split(queryEdgeType, ",")
	}

	// Find path
	result := traverser.FindPath(from, to, edgeTypes)

	if len(result.Visited) == 0 {
		output := map[string]any{
			"from":    from,
			"to":      to,
			"found":   false,
			"message": "No path found between nodes",
		}
		return printOutput(output)
	}

	// Build edges output
	var edgesOut []map[string]any
	for _, e := range result.Edges {
		edgesOut = append(edgesOut, map[string]any{
			"from": e.From,
			"to":   e.To,
			"type": e.Type,
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

func listEdges(graphStore *store.FSStore, args []string) error {
	// Load edges
	edges, err := graphStore.ListEdges()
	if err != nil {
		return fmt.Errorf("loading edges: %w", err)
	}

	// Filter edges
	var matches []map[string]any
	nodeID := ""
	if len(args) > 0 {
		nodeID = args[0]
	}

	for _, e := range edges {
		// Filter by node ID
		if nodeID != "" && e.From != nodeID && e.To != nodeID {
			continue
		}

		// Filter by --from
		if queryFrom != "" && e.From != queryFrom {
			continue
		}

		// Filter by --type
		if queryType != "" && e.Type != queryType {
			continue
		}

		matches = append(matches, map[string]any{
			"from":       e.From,
			"to":         e.To,
			"type":       e.Type,
			"confidence": e.Confidence,
		})

		if queryLimit > 0 && len(matches) >= queryLimit {
			break
		}
	}

	result := map[string]any{
		"query":   nodeID,
		"matches": len(matches),
		"edges":   matches,
	}

	if queryLimit > 0 && len(matches) >= queryLimit {
		result["truncated"] = true
		result["message"] = fmt.Sprintf("Showing first %d matches. Use --limit to increase.", queryLimit)
	}

	return printOutput(result)
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

	// Count by type
	nodeTypes := make(map[string]int)
	for _, n := range nodes {
		nodeTypes[n.Type]++
	}

	edgeTypes := make(map[string]int)
	for _, e := range edges {
		edgeTypes[e.Type]++
	}

	// Find top nodes by edge count
	edgeCounts := make(map[string]int)
	for _, e := range edges {
		edgeCounts[e.From]++
		edgeCounts[e.To]++
	}

	type nodeCount struct {
		ID    string
		Count int
	}
	var topNodes []nodeCount
	for id, count := range edgeCounts {
		topNodes = append(topNodes, nodeCount{id, count})
	}
	sort.Slice(topNodes, func(i, j int) bool {
		return topNodes[i].Count > topNodes[j].Count
	})

	// Take top 10, filtering out less interesting nodes
	var godNodes []map[string]any
	for _, n := range topNodes {
		// Skip external packages and generic types
		if strings.HasPrefix(n.ID, "pkg_") && strings.Contains(n.ID, "/") {
			continue
		}
		if strings.HasPrefix(n.ID, "call_") {
			continue
		}
		godNodes = append(godNodes, map[string]any{
			"id":    n.ID,
			"edges": n.Count,
		})
		if len(godNodes) >= 10 {
			break
		}
	}

	result := map[string]any{
		"total_nodes": len(nodes),
		"total_edges": len(edges),
		"node_types":  nodeTypes,
		"edge_types":  edgeTypes,
		"god_nodes":   godNodes,
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
