// Package analyze provides graph analysis functions.
package analyze

import (
	"sort"

	"github.com/plexusone/graphfs/pkg/graph"
	"gonum.org/v1/gonum/graph/network"
	"gonum.org/v1/gonum/graph/simple"
)

// BridgeNode represents a node with high betweenness centrality.
// These nodes act as bridges connecting different parts of the graph.
type BridgeNode struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Type       string  `json:"type"`
	Centrality float64 `json:"centrality"`
	// Communities this node connects (if community detection was run)
	ConnectsCommunities []int `json:"connects_communities,omitempty"`
}

// BetweennessResult contains the results of betweenness centrality analysis.
type BetweennessResult struct {
	// Bridges are nodes with highest betweenness centrality
	Bridges []BridgeNode `json:"bridges"`
	// Scores maps node ID to its betweenness centrality score
	Scores map[string]float64 `json:"scores"`
	// MaxScore is the highest betweenness score in the graph
	MaxScore float64 `json:"max_score"`
}

// BetweennessOptions configures betweenness centrality calculation.
type BetweennessOptions struct {
	// TopN limits results to top N bridges (0 = all)
	TopN int
	// ExcludeNodeTypes filters out these node types from results
	ExcludeNodeTypes []string
	// ExcludeEdgeTypes filters out these edge types from the graph
	ExcludeEdgeTypes []string
	// Communities provides community membership for bridge analysis
	Communities map[int][]string
}

// DefaultBetweennessOptions returns sensible defaults.
func DefaultBetweennessOptions() BetweennessOptions {
	return BetweennessOptions{
		TopN:             20,
		ExcludeNodeTypes: []string{"package", "file"},
		ExcludeEdgeTypes: []string{"contains"},
	}
}

// CalculateBetweenness computes betweenness centrality for all nodes.
// Betweenness centrality measures how often a node lies on shortest paths
// between other nodes. High betweenness indicates architectural bridges.
func CalculateBetweenness(nodes []*graph.Node, edges []*graph.Edge, opts BetweennessOptions) *BetweennessResult {
	if len(nodes) == 0 {
		return &BetweennessResult{
			Bridges: []BridgeNode{},
			Scores:  make(map[string]float64),
		}
	}

	// Build exclusion sets
	excludeNodeTypes := make(map[string]bool)
	for _, t := range opts.ExcludeNodeTypes {
		excludeNodeTypes[t] = true
	}
	excludeEdgeTypes := make(map[string]bool)
	for _, t := range opts.ExcludeEdgeTypes {
		excludeEdgeTypes[t] = true
	}

	// Build node map and ID mapping for gonum
	nodeMap := make(map[string]*graph.Node)
	nodeToID := make(map[string]int64)
	idToNode := make(map[int64]string)
	var nextID int64

	for _, n := range nodes {
		nodeMap[n.ID] = n
		nodeToID[n.ID] = nextID
		idToNode[nextID] = n.ID
		nextID++
	}

	// Build gonum graph
	g := simple.NewUndirectedGraph()

	// Add nodes
	for _, n := range nodes {
		g.AddNode(simple.Node(nodeToID[n.ID]))
	}

	// Add edges (excluding filtered types)
	for _, e := range edges {
		if excludeEdgeTypes[e.Type] {
			continue
		}
		fromID, fromOK := nodeToID[e.From]
		toID, toOK := nodeToID[e.To]
		if fromOK && toOK && fromID != toID {
			// Avoid self-loops and duplicate edges
			if g.Edge(fromID, toID) == nil {
				g.SetEdge(simple.Edge{F: simple.Node(fromID), T: simple.Node(toID)})
			}
		}
	}

	// Calculate betweenness centrality
	betweenness := network.Betweenness(g)

	// Build result
	result := &BetweennessResult{
		Scores: make(map[string]float64),
	}

	// Convert gonum IDs back to node IDs
	for id, score := range betweenness {
		nodeID := idToNode[id]
		result.Scores[nodeID] = score
		if score > result.MaxScore {
			result.MaxScore = score
		}
	}

	// Build community membership map (node -> communities)
	nodeToCommunities := make(map[string][]int)
	if opts.Communities != nil {
		for commID, members := range opts.Communities {
			for _, nodeID := range members {
				nodeToCommunities[nodeID] = append(nodeToCommunities[nodeID], commID)
			}
		}
	}

	// Build adjacency for community connection detection
	neighbors := make(map[string]map[string]bool)
	for _, e := range edges {
		if excludeEdgeTypes[e.Type] {
			continue
		}
		if neighbors[e.From] == nil {
			neighbors[e.From] = make(map[string]bool)
		}
		if neighbors[e.To] == nil {
			neighbors[e.To] = make(map[string]bool)
		}
		neighbors[e.From][e.To] = true
		neighbors[e.To][e.From] = true
	}

	// Sort nodes by betweenness score
	type nodeScore struct {
		id    string
		score float64
	}
	var sorted []nodeScore
	for nodeID, score := range result.Scores {
		node := nodeMap[nodeID]
		if node == nil || excludeNodeTypes[node.Type] {
			continue
		}
		if score > 0 {
			sorted = append(sorted, nodeScore{nodeID, score})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].score > sorted[j].score
	})

	// Take top N
	limit := len(sorted)
	if opts.TopN > 0 && opts.TopN < limit {
		limit = opts.TopN
	}

	// Build bridges list
	for i := 0; i < limit; i++ {
		nodeID := sorted[i].id
		node := nodeMap[nodeID]

		bridge := BridgeNode{
			ID:         nodeID,
			Label:      node.Label,
			Type:       node.Type,
			Centrality: sorted[i].score,
		}

		if bridge.Label == "" {
			bridge.Label = nodeID
		}

		// Find which communities this node connects
		if opts.Communities != nil {
			connectedComms := make(map[int]bool)
			// Add node's own communities
			for _, c := range nodeToCommunities[nodeID] {
				connectedComms[c] = true
			}
			// Add communities of neighbors
			for neighbor := range neighbors[nodeID] {
				for _, c := range nodeToCommunities[neighbor] {
					connectedComms[c] = true
				}
			}
			// Only include if connecting multiple communities
			if len(connectedComms) > 1 {
				for c := range connectedComms {
					bridge.ConnectsCommunities = append(bridge.ConnectsCommunities, c)
				}
				sort.Ints(bridge.ConnectsCommunities)
			}
		}

		result.Bridges = append(result.Bridges, bridge)
	}

	return result
}

// FindBridges is a convenience function that finds architectural bridges
// using default options.
func FindBridges(nodes []*graph.Node, edges []*graph.Edge, topN int) []BridgeNode {
	opts := DefaultBetweennessOptions()
	opts.TopN = topN
	result := CalculateBetweenness(nodes, edges, opts)
	return result.Bridges
}

// FindBridgesWithCommunities finds bridges and annotates which communities they connect.
func FindBridgesWithCommunities(nodes []*graph.Node, edges []*graph.Edge, communities map[int][]string, topN int) []BridgeNode {
	opts := DefaultBetweennessOptions()
	opts.TopN = topN
	opts.Communities = communities
	result := CalculateBetweenness(nodes, edges, opts)
	return result.Bridges
}
