// Package analyze provides graph analysis utilities.
package analyze

import "github.com/plexusone/graphfs/pkg/graph"

// GroupEdgesByType groups edges by their Type field.
func GroupEdgesByType(edges []*graph.Edge) map[string][]*graph.Edge {
	result := make(map[string][]*graph.Edge)
	for _, e := range edges {
		result[e.Type] = append(result[e.Type], e)
	}
	return result
}

// CountEdgesByType returns edge counts per type.
func CountEdgesByType(edges []*graph.Edge) map[string]int {
	result := make(map[string]int)
	for _, e := range edges {
		result[e.Type]++
	}
	return result
}

// GroupEdgesByConfidence groups edges by their Confidence field.
func GroupEdgesByConfidence(edges []*graph.Edge) map[string][]*graph.Edge {
	result := make(map[string][]*graph.Edge)
	for _, e := range edges {
		result[string(e.Confidence)] = append(result[string(e.Confidence)], e)
	}
	return result
}

// CountEdgesByConfidence returns edge counts per confidence level.
func CountEdgesByConfidence(edges []*graph.Edge) map[string]int {
	result := make(map[string]int)
	for _, e := range edges {
		result[string(e.Confidence)]++
	}
	return result
}
