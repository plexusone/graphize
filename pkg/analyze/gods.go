// Package analyze provides graph analysis features including god nodes,
// community detection, and surprising connections.
package analyze

import (
	"sort"
	"strings"

	"github.com/plexusone/graphfs/pkg/analyze"
	"github.com/plexusone/graphfs/pkg/graph"
)

// Re-export generic types from graphfs for backward compatibility
type (
	HubNode   = analyze.HubNode
	Community = analyze.Community
)

// Re-export generic functions from graphfs
var (
	NodesByType       = analyze.NodesByType
	EdgesByType       = analyze.EdgesByType
	EdgesByConfidence = analyze.EdgesByConfidence
	HubScore          = analyze.HubScore
	AuthorityScore    = analyze.AuthorityScore
	InferredEdges     = analyze.InferredEdges
	CohesionScore     = analyze.CohesionScore
)

// GodNode is an alias for HubNode with code-specific naming.
// Deprecated: Use HubNode instead.
type GodNode = HubNode

// GodNodes returns the top N most connected nodes in the graph.
// Excludes file-level hub nodes (packages, files) to focus on
// meaningful architectural abstractions like functions and types.
func GodNodes(nodes []*graph.Node, edges []*graph.Edge, topN int) []GodNode {
	// Use graphfs FindHubs with code-specific exclusions
	return analyze.FindHubs(nodes, edges, topN, []string{
		graph.NodeTypePackage,
		graph.NodeTypeFile,
	})
}

// isFileHub returns true for nodes that are structural hubs (packages, files)
// rather than meaningful code entities.
func isFileHub(n *graph.Node) bool {
	switch n.Type {
	case graph.NodeTypePackage, graph.NodeTypeFile:
		return true
	}
	// Also skip external packages (imports)
	if n.Attrs != nil && n.Attrs["external"] == "true" {
		return true
	}
	return false
}

// IsolatedNodes returns nodes with degree <= threshold.
// These represent potential documentation gaps or orphaned code.
func IsolatedNodes(nodes []*graph.Node, edges []*graph.Edge, threshold int) []*graph.Node {
	// Use graphfs IsolatedNodes with code-specific exclusions
	return analyze.IsolatedNodes(nodes, edges, threshold, []string{
		graph.NodeTypePackage,
		graph.NodeTypeFile,
	})
}

// CrossFileEdges returns edges that connect nodes in different source files.
func CrossFileEdges(nodes []*graph.Node, edges []*graph.Edge) []*graph.Edge {
	// Build node -> source file map
	nodeFile := make(map[string]string)
	for _, n := range nodes {
		if n.Attrs != nil {
			nodeFile[n.ID] = n.Attrs["source_file"]
		}
	}

	var crossFile []*graph.Edge
	for _, e := range edges {
		fromFile := nodeFile[e.From]
		toFile := nodeFile[e.To]
		if fromFile != "" && toFile != "" && fromFile != toFile {
			crossFile = append(crossFile, e)
		}
	}

	return crossFile
}

// PackageStats returns statistics about packages in the graph.
type PackageStats struct {
	Name      string `json:"name"`
	Files     int    `json:"files"`
	Functions int    `json:"functions"`
	Types     int    `json:"types"`
	Imports   int    `json:"imports"`
}

// AnalyzePackages returns statistics for each package in the graph.
func AnalyzePackages(nodes []*graph.Node, edges []*graph.Edge) []PackageStats {
	// Group nodes by package
	packages := make(map[string]*PackageStats)

	for _, n := range nodes {
		pkg := ""
		if n.Attrs != nil {
			pkg = n.Attrs["package"]
		}
		if pkg == "" && n.Type == graph.NodeTypePackage {
			pkg = n.ID
		}
		if pkg == "" {
			continue
		}

		// Clean package name
		pkg = strings.TrimPrefix(pkg, "pkg_")

		if packages[pkg] == nil {
			packages[pkg] = &PackageStats{Name: pkg}
		}

		switch n.Type {
		case graph.NodeTypeFile:
			packages[pkg].Files++
		case graph.NodeTypeFunction, graph.NodeTypeMethod:
			packages[pkg].Functions++
		case graph.NodeTypeStruct, graph.NodeTypeInterface:
			packages[pkg].Types++
		}
	}

	// Count imports per package
	for _, e := range edges {
		if e.Type == graph.EdgeTypeImports {
			// Find the package of the source node
			for _, n := range nodes {
				if n.ID == e.From && n.Attrs != nil {
					pkg := strings.TrimPrefix(n.Attrs["package"], "pkg_")
					if packages[pkg] != nil {
						packages[pkg].Imports++
					}
					break
				}
			}
		}
	}

	// Convert to slice
	var result []PackageStats
	for _, stats := range packages {
		result = append(result, *stats)
	}

	// Sort by function count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Functions > result[j].Functions
	})

	return result
}
