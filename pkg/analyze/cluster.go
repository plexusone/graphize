// Package analyze provides graph analysis functions.
package analyze

import (
	"github.com/plexusone/graphfs/pkg/analyze"
	"github.com/plexusone/graphfs/pkg/graph"
)

// Re-export ClusterResult from graphfs for backward compatibility
type ClusterResult = analyze.ClusterResult

// Re-export cluster options from graphfs
type (
	ClusterOptions   = analyze.ClusterOptions
	ClusterAlgorithm = analyze.ClusterAlgorithm
	LouvainOptions   = analyze.LouvainOptions
	LouvainResult    = analyze.LouvainResult
)

// Re-export algorithm constants
const (
	AlgorithmLouvain             = analyze.AlgorithmLouvain
	AlgorithmConnectedComponents = analyze.AlgorithmConnectedComponents
)

// Re-export cluster functions
var (
	DefaultClusterOptions    = analyze.DefaultClusterOptions
	DefaultLouvainOptions    = analyze.DefaultLouvainOptions
	DetectCommunitiesLouvain = analyze.DetectCommunitiesLouvain
	LouvainToClusters        = analyze.LouvainToClusters
)

// DetectCommunities performs community detection using the Louvain algorithm.
func DetectCommunities(nodes []*graph.Node, edges []*graph.Edge) *ClusterResult {
	return analyze.DetectCommunities(nodes, edges)
}

// DetectCommunitiesWithOptions performs community detection with configurable options.
func DetectCommunitiesWithOptions(nodes []*graph.Node, edges []*graph.Edge, opts ClusterOptions) *ClusterResult {
	return analyze.DetectCommunitiesWithOptions(nodes, edges, opts)
}

// CommunityLabels generates human-readable labels for communities based on member nodes.
func CommunityLabels(communities []Community, nodes []*graph.Node) map[int]string {
	// Build node map
	nodeMap := make(map[string]*graph.Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	labels := make(map[int]string)
	for _, c := range communities {
		label := ""
		// Try to find a package node first
		for _, m := range c.Members {
			if n, ok := nodeMap[m]; ok && n.Type == graph.NodeTypePackage {
				label = n.Label
				if label == "" {
					label = n.ID
				}
				break
			}
		}
		// Fall back to first function or type
		if label == "" {
			for _, m := range c.Members {
				if n, ok := nodeMap[m]; ok {
					if n.Type == graph.NodeTypeFunction || n.Type == graph.NodeTypeStruct {
						label = n.Label
						if label == "" {
							label = n.ID
						}
						break
					}
				}
			}
		}
		// Final fallback
		if label == "" && len(c.Members) > 0 {
			if n, ok := nodeMap[c.Members[0]]; ok {
				label = n.Label
				if label == "" {
					label = n.ID
				}
			}
		}
		labels[c.ID] = label
	}
	return labels
}
