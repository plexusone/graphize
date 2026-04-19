// Package query provides graph query utilities and result formatting.
package query

import (
	"sort"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
)

// Summary contains aggregate statistics about a graph.
type Summary struct {
	TotalNodes int            `json:"total_nodes"`
	TotalEdges int            `json:"total_edges"`
	NodeTypes  map[string]int `json:"node_types"`
	EdgeTypes  map[string]int `json:"edge_types"`
	GodNodes   []GodNode      `json:"god_nodes"`
}

// GodNode represents a highly-connected node in the graph.
type GodNode struct {
	ID        string `json:"id"`
	EdgeCount int    `json:"edges"`
}

// ComputeSummary calculates summary statistics for a graph.
func ComputeSummary(g *graph.Graph) *Summary {
	return ComputeSummaryWithOptions(g, DefaultSummaryOptions())
}

// SummaryOptions controls summary computation behavior.
type SummaryOptions struct {
	// MaxGodNodes is the maximum number of god nodes to return.
	MaxGodNodes int

	// ExcludePrefixes are node ID prefixes to exclude from god nodes.
	ExcludePrefixes []string

	// ExcludeContains are substrings that exclude nodes from god nodes.
	ExcludeContains []string
}

// DefaultSummaryOptions returns sensible defaults for summary computation.
func DefaultSummaryOptions() SummaryOptions {
	return SummaryOptions{
		MaxGodNodes:     10,
		ExcludePrefixes: []string{"call_"},
		ExcludeContains: []string{},
	}
}

// ComputeSummaryWithOptions calculates summary with custom options.
func ComputeSummaryWithOptions(g *graph.Graph, opts SummaryOptions) *Summary {
	// Count nodes by type
	nodeTypes := make(map[string]int)
	for _, n := range g.Nodes {
		nodeTypes[n.Type]++
	}

	// Count edges by type
	edgeTypes := make(map[string]int)
	for _, e := range g.Edges {
		edgeTypes[e.Type]++
	}

	// Count edges per node
	edgeCounts := make(map[string]int)
	for _, e := range g.Edges {
		edgeCounts[e.From]++
		edgeCounts[e.To]++
	}

	// Sort by edge count
	type nodeCount struct {
		ID    string
		Count int
	}
	var sorted []nodeCount
	for id, count := range edgeCounts {
		sorted = append(sorted, nodeCount{id, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	// Filter and collect god nodes
	var godNodes []GodNode
	for _, n := range sorted {
		if shouldExcludeNode(n.ID, opts) {
			continue
		}
		godNodes = append(godNodes, GodNode{
			ID:        n.ID,
			EdgeCount: n.Count,
		})
		if len(godNodes) >= opts.MaxGodNodes {
			break
		}
	}

	return &Summary{
		TotalNodes: len(g.Nodes),
		TotalEdges: len(g.Edges),
		NodeTypes:  nodeTypes,
		EdgeTypes:  edgeTypes,
		GodNodes:   godNodes,
	}
}

func shouldExcludeNode(id string, opts SummaryOptions) bool {
	for _, prefix := range opts.ExcludePrefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	for _, substr := range opts.ExcludeContains {
		if strings.Contains(id, substr) {
			return true
		}
	}
	return false
}

// ComputeSummaryFromLists calculates summary from node and edge lists.
// This is useful when you have lists but not a full graph.
func ComputeSummaryFromLists(nodes []*graph.Node, edges []*graph.Edge) *Summary {
	// Build temporary graph
	g := &graph.Graph{
		Nodes: make(map[string]*graph.Node),
		Edges: edges,
	}
	for _, n := range nodes {
		g.Nodes[n.ID] = n
	}
	return ComputeSummary(g)
}
