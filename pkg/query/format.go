package query

import (
	"sort"

	gquery "github.com/plexusone/graphfs/pkg/query"
)

// TraversalOutput represents formatted traversal results.
type TraversalOutput struct {
	Query      string       `json:"query"`
	Algorithm  string       `json:"algorithm"`
	Direction  string       `json:"direction"`
	MaxDepth   int          `json:"max_depth"`
	NodesFound int          `json:"nodes_found"`
	EdgesFound int          `json:"edges_found"`
	Layers     []DepthLayer `json:"layers"`
	Edges      []EdgeOutput `json:"edges"`
}

// DepthLayer represents nodes at a specific depth in traversal.
type DepthLayer struct {
	Depth int      `json:"depth"`
	Count int      `json:"count"`
	Nodes []string `json:"nodes"`
}

// EdgeOutput represents a simplified edge for output.
type EdgeOutput struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// FormatTraversalOptions controls traversal output formatting.
type FormatTraversalOptions struct {
	// Limit is the maximum nodes/edges to include per layer.
	Limit int

	// Algorithm is "BFS" or "DFS".
	Algorithm string

	// Direction is the traversal direction.
	Direction string
}

// FormatTraversal converts a TraversalResult to formatted output.
func FormatTraversal(result *gquery.TraversalResult, startNode string, maxDepth int, opts FormatTraversalOptions) *TraversalOutput {
	// Group nodes by depth
	nodesByDepth := make(map[int][]string)
	for node, depth := range result.Depth {
		nodesByDepth[depth] = append(nodesByDepth[depth], node)
	}

	// Build depth layers
	var layers []DepthLayer
	for d := 0; d <= maxDepth; d++ {
		nodes := nodesByDepth[d]
		if len(nodes) == 0 {
			continue
		}
		sort.Strings(nodes)

		displayNodes := nodes
		if opts.Limit > 0 && len(displayNodes) > opts.Limit {
			displayNodes = displayNodes[:opts.Limit]
		}

		layers = append(layers, DepthLayer{
			Depth: d,
			Count: len(nodes),
			Nodes: displayNodes,
		})
	}

	// Build edges output
	var edges []EdgeOutput
	for _, e := range result.Edges {
		edges = append(edges, EdgeOutput{
			From: e.From,
			To:   e.To,
			Type: e.Type,
		})
	}
	if opts.Limit > 0 && len(edges) > opts.Limit {
		edges = edges[:opts.Limit]
	}

	return &TraversalOutput{
		Query:      startNode,
		Algorithm:  opts.Algorithm,
		Direction:  opts.Direction,
		MaxDepth:   maxDepth,
		NodesFound: len(result.Visited),
		EdgesFound: len(result.Edges),
		Layers:     layers,
		Edges:      edges,
	}
}

// PathOutput represents formatted path-finding results.
type PathOutput struct {
	From    string       `json:"from"`
	To      string       `json:"to"`
	Found   bool         `json:"found"`
	Length  int          `json:"length,omitempty"`
	Path    []string     `json:"path,omitempty"`
	Edges   []EdgeOutput `json:"edges,omitempty"`
	Message string       `json:"message,omitempty"`
}

// FormatPath converts a path-finding result to formatted output.
func FormatPath(result *gquery.TraversalResult, from, to string) *PathOutput {
	if len(result.Visited) == 0 {
		return &PathOutput{
			From:    from,
			To:      to,
			Found:   false,
			Message: "No path found between nodes",
		}
	}

	var edges []EdgeOutput
	for _, e := range result.Edges {
		edges = append(edges, EdgeOutput{
			From: e.From,
			To:   e.To,
			Type: e.Type,
		})
	}

	return &PathOutput{
		From:   from,
		To:     to,
		Found:  true,
		Length: len(result.Visited) - 1,
		Path:   result.Visited,
		Edges:  edges,
	}
}
