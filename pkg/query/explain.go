package query

import (
	"fmt"
	"sort"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/pkg/analyze"
)

// NodeExplanation provides comprehensive context about a node.
type NodeExplanation struct {
	// Node is the basic node information.
	Node NodeInfo `json:"node"`

	// Neighbors describes the node's immediate connections.
	Neighbors NeighborInfo `json:"neighbors"`

	// Community describes which community the node belongs to.
	Community CommunityInfo `json:"community"`

	// Centrality describes the node's centrality metrics.
	Centrality CentralityInfo `json:"centrality"`

	// SourceFile is the file containing this node (if applicable).
	SourceFile string `json:"source_file,omitempty"`

	// Package is the package this node belongs to (if applicable).
	Package string `json:"package,omitempty"`
}

// NodeInfo contains basic node metadata.
type NodeInfo struct {
	ID    string            `json:"id"`
	Type  string            `json:"type"`
	Label string            `json:"label"`
	Attrs map[string]string `json:"attrs,omitempty"`
}

// NeighborInfo describes a node's connections.
type NeighborInfo struct {
	// InDegree is the number of incoming edges.
	InDegree int `json:"in_degree"`

	// OutDegree is the number of outgoing edges.
	OutDegree int `json:"out_degree"`

	// TotalDegree is in + out degree.
	TotalDegree int `json:"total_degree"`

	// IncomingEdges lists edges pointing to this node.
	IncomingEdges []EdgeInfo `json:"incoming_edges,omitempty"`

	// OutgoingEdges lists edges from this node.
	OutgoingEdges []EdgeInfo `json:"outgoing_edges,omitempty"`

	// EdgeTypeBreakdown shows count by edge type.
	EdgeTypeBreakdown map[string]int `json:"edge_type_breakdown"`
}

// EdgeInfo describes an edge for explanation.
type EdgeInfo struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	Confidence string `json:"confidence,omitempty"`
}

// CommunityInfo describes community membership.
type CommunityInfo struct {
	// CommunityID is the cluster ID this node belongs to.
	CommunityID int `json:"community_id"`

	// CommunityLabel is a human-readable label for the community.
	CommunityLabel string `json:"community_label"`

	// CommunitySize is the number of nodes in the community.
	CommunitySize int `json:"community_size"`

	// IsBridge indicates if this node connects multiple communities.
	IsBridge bool `json:"is_bridge"`

	// BridgesCommunities lists communities this node connects (if bridge).
	BridgesCommunities []int `json:"bridges_communities,omitempty"`
}

// CentralityInfo describes centrality metrics.
type CentralityInfo struct {
	// Betweenness is the betweenness centrality score.
	Betweenness float64 `json:"betweenness"`

	// BetweennessRank is the rank among all nodes (1 = highest).
	BetweennessRank int `json:"betweenness_rank"`

	// IsHub indicates if this is a highly connected node.
	IsHub bool `json:"is_hub"`

	// IsBridge indicates if this is a betweenness bridge.
	IsBridge bool `json:"is_bridge"`
}

// ExplainOptions configures the explain operation.
type ExplainOptions struct {
	// Depth controls how many levels of neighbors to include.
	Depth int

	// MaxNeighbors limits the number of neighbors shown.
	MaxNeighbors int

	// IncludeEdges includes edge details in output.
	IncludeEdges bool
}

// DefaultExplainOptions returns sensible defaults.
func DefaultExplainOptions() ExplainOptions {
	return ExplainOptions{
		Depth:        1,
		MaxNeighbors: 20,
		IncludeEdges: true,
	}
}

// ExplainNode provides comprehensive context about a specific node.
func ExplainNode(nodes []*graph.Node, edges []*graph.Edge, nodeID string, opts ExplainOptions) (*NodeExplanation, error) {
	// Find the target node
	var targetNode *graph.Node
	nodeMap := make(map[string]*graph.Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
		if n.ID == nodeID {
			targetNode = n
		}
	}

	if targetNode == nil {
		return nil, fmt.Errorf("node %q not found", nodeID)
	}

	explanation := &NodeExplanation{
		Node: NodeInfo{
			ID:    targetNode.ID,
			Type:  targetNode.Type,
			Label: targetNode.Label,
			Attrs: targetNode.Attrs,
		},
	}

	// Extract source file and package from attrs
	if targetNode.Attrs != nil {
		explanation.SourceFile = targetNode.Attrs["source_file"]
		explanation.Package = targetNode.Attrs["package"]
	}

	// Compute neighbor information
	explanation.Neighbors = computeNeighborInfo(edges, nodeID, opts)

	// Compute community information
	explanation.Community = computeCommunityInfo(nodes, edges, nodeID)

	// Compute centrality information
	explanation.Centrality = computeCentralityInfo(nodes, edges, nodeID)

	return explanation, nil
}

// computeNeighborInfo analyzes edges connecting to a node.
func computeNeighborInfo(edges []*graph.Edge, nodeID string, opts ExplainOptions) NeighborInfo {
	info := NeighborInfo{
		EdgeTypeBreakdown: make(map[string]int),
	}

	for _, e := range edges {
		if e.From == nodeID {
			info.OutDegree++
			info.EdgeTypeBreakdown[e.Type]++
			if opts.IncludeEdges && len(info.OutgoingEdges) < opts.MaxNeighbors {
				info.OutgoingEdges = append(info.OutgoingEdges, EdgeInfo{
					From:       e.From,
					To:         e.To,
					Type:       e.Type,
					Confidence: string(e.Confidence),
				})
			}
		}
		if e.To == nodeID {
			info.InDegree++
			info.EdgeTypeBreakdown[e.Type]++
			if opts.IncludeEdges && len(info.IncomingEdges) < opts.MaxNeighbors {
				info.IncomingEdges = append(info.IncomingEdges, EdgeInfo{
					From:       e.From,
					To:         e.To,
					Type:       e.Type,
					Confidence: string(e.Confidence),
				})
			}
		}
	}

	info.TotalDegree = info.InDegree + info.OutDegree
	return info
}

// computeCommunityInfo determines community membership.
func computeCommunityInfo(nodes []*graph.Node, edges []*graph.Edge, nodeID string) CommunityInfo {
	// Run community detection
	clusterResult := analyze.DetectCommunities(nodes, edges)

	info := CommunityInfo{
		CommunityID: -1,
	}

	// Find which community this node belongs to
	nodeToCommunity := make(map[string]int)
	for _, c := range clusterResult.Communities {
		for _, member := range c.Members {
			nodeToCommunity[member] = c.ID
			if member == nodeID {
				info.CommunityID = c.ID
				info.CommunitySize = c.Size
			}
		}
	}

	// Generate community label
	if info.CommunityID >= 0 {
		labels := analyze.CommunityLabels(clusterResult.Communities, nodes)
		info.CommunityLabel = labels[info.CommunityID]
	}

	// Check if this is a bridge node (connects multiple communities)
	connectedCommunities := make(map[int]bool)
	if info.CommunityID >= 0 {
		connectedCommunities[info.CommunityID] = true
	}

	for _, e := range edges {
		var neighborID string
		if e.From == nodeID {
			neighborID = e.To
		} else if e.To == nodeID {
			neighborID = e.From
		} else {
			continue
		}

		if neighborComm, ok := nodeToCommunity[neighborID]; ok {
			connectedCommunities[neighborComm] = true
		}
	}

	if len(connectedCommunities) > 1 {
		info.IsBridge = true
		for commID := range connectedCommunities {
			info.BridgesCommunities = append(info.BridgesCommunities, commID)
		}
		sort.Ints(info.BridgesCommunities)
	}

	return info
}

// computeCentralityInfo calculates centrality metrics for a node.
func computeCentralityInfo(nodes []*graph.Node, edges []*graph.Edge, nodeID string) CentralityInfo {
	// Calculate betweenness centrality
	betweennessResult := analyze.CalculateBetweenness(nodes, edges, analyze.DefaultBetweennessOptions())

	info := CentralityInfo{}

	// Get this node's score
	if score, ok := betweennessResult.Scores[nodeID]; ok {
		info.Betweenness = score
	}

	// Calculate rank
	type scoreRank struct {
		id    string
		score float64
	}
	var scores []scoreRank
	for id, score := range betweennessResult.Scores {
		scores = append(scores, scoreRank{id, score})
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	for i, sr := range scores {
		if sr.id == nodeID {
			info.BetweennessRank = i + 1
			break
		}
	}

	// Determine if this is a hub (top 10% by betweenness)
	topCount := len(scores) / 10
	if topCount < 1 {
		topCount = 1
	}
	if info.BetweennessRank > 0 && info.BetweennessRank <= topCount {
		info.IsBridge = true
	}

	// Check hub status based on degree
	for _, b := range betweennessResult.Bridges {
		if b.ID == nodeID {
			info.IsBridge = true
			break
		}
	}

	return info
}

// FormatExplanation formats the explanation as human-readable text.
func FormatExplanation(e *NodeExplanation) string {
	var sb strings.Builder

	// Header
	label := e.Node.Label
	if label == "" {
		label = e.Node.ID
	}
	fmt.Fprintf(&sb, "# %s\n\n", label)
	fmt.Fprintf(&sb, "**Type:** %s\n", e.Node.Type)
	fmt.Fprintf(&sb, "**ID:** `%s`\n", e.Node.ID)

	if e.SourceFile != "" {
		fmt.Fprintf(&sb, "**File:** %s\n", e.SourceFile)
	}
	if e.Package != "" {
		fmt.Fprintf(&sb, "**Package:** %s\n", e.Package)
	}

	sb.WriteString("\n")

	// Connectivity
	sb.WriteString("## Connectivity\n\n")
	fmt.Fprintf(&sb, "- **Total Degree:** %d (%d incoming, %d outgoing)\n",
		e.Neighbors.TotalDegree, e.Neighbors.InDegree, e.Neighbors.OutDegree)

	if len(e.Neighbors.EdgeTypeBreakdown) > 0 {
		sb.WriteString("- **Edge Types:**\n")
		for edgeType, count := range e.Neighbors.EdgeTypeBreakdown {
			fmt.Fprintf(&sb, "  - %s: %d\n", edgeType, count)
		}
	}
	sb.WriteString("\n")

	// Community
	sb.WriteString("## Community\n\n")
	if e.Community.CommunityID >= 0 {
		fmt.Fprintf(&sb, "- **Community ID:** %d\n", e.Community.CommunityID)
		if e.Community.CommunityLabel != "" {
			fmt.Fprintf(&sb, "- **Community Label:** %s\n", e.Community.CommunityLabel)
		}
		fmt.Fprintf(&sb, "- **Community Size:** %d nodes\n", e.Community.CommunitySize)
		if e.Community.IsBridge {
			fmt.Fprintf(&sb, "- **Bridge Node:** Connects communities %v\n", e.Community.BridgesCommunities)
		}
	} else {
		sb.WriteString("- *No community assigned*\n")
	}
	sb.WriteString("\n")

	// Centrality
	sb.WriteString("## Centrality\n\n")
	fmt.Fprintf(&sb, "- **Betweenness Score:** %.2f\n", e.Centrality.Betweenness)
	if e.Centrality.BetweennessRank > 0 {
		fmt.Fprintf(&sb, "- **Betweenness Rank:** #%d\n", e.Centrality.BetweennessRank)
	}
	if e.Centrality.IsBridge {
		sb.WriteString("- **Architectural Bridge:** Yes (high betweenness)\n")
	}
	sb.WriteString("\n")

	// Edges
	if len(e.Neighbors.OutgoingEdges) > 0 {
		sb.WriteString("## Outgoing Edges\n\n")
		sb.WriteString("| To | Type | Confidence |\n")
		sb.WriteString("|----|------|------------|\n")
		for _, edge := range e.Neighbors.OutgoingEdges {
			fmt.Fprintf(&sb, "| %s | %s | %s |\n", edge.To, edge.Type, edge.Confidence)
		}
		sb.WriteString("\n")
	}

	if len(e.Neighbors.IncomingEdges) > 0 {
		sb.WriteString("## Incoming Edges\n\n")
		sb.WriteString("| From | Type | Confidence |\n")
		sb.WriteString("|------|------|------------|\n")
		for _, edge := range e.Neighbors.IncomingEdges {
			fmt.Fprintf(&sb, "| %s | %s | %s |\n", edge.From, edge.Type, edge.Confidence)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ExplainNodeInGraph provides explanation using a Graph object.
func ExplainNodeInGraph(g *graph.Graph, nodeID string, opts ExplainOptions) (*NodeExplanation, error) {
	// Convert graph to lists
	var nodes []*graph.Node
	for _, n := range g.Nodes {
		nodes = append(nodes, n)
	}
	return ExplainNode(nodes, g.Edges, nodeID, opts)
}
