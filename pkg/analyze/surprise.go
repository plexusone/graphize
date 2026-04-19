package analyze

import (
	"sort"

	"github.com/plexusone/graphfs/pkg/graph"
)

// Surprise represents a surprising connection in the graph.
type Surprise struct {
	From            string   `json:"from"`
	FromLabel       string   `json:"from_label"`
	To              string   `json:"to"`
	ToLabel         string   `json:"to_label"`
	Type            string   `json:"type"`
	Confidence      string   `json:"confidence"`
	Score           float64  `json:"score"`
	ConfidenceScore float64  `json:"confidence_score,omitempty"`
	Why             string   `json:"why"`
	SourceFiles     []string `json:"source_files,omitempty"`
}

// SurprisingConnections finds edges that are unexpected or non-obvious.
// Prioritizes:
// 1. AMBIGUOUS edges (uncertain relationships)
// 2. INFERRED edges (LLM-inferred relationships)
// 3. Cross-file edges between unrelated components
func SurprisingConnections(nodes []*graph.Node, edges []*graph.Edge, communities map[int][]string, topN int) []Surprise {
	// Build node maps
	nodeMap := make(map[string]*graph.Node)
	nodeFile := make(map[string]string)
	for _, n := range nodes {
		nodeMap[n.ID] = n
		if n.Attrs != nil {
			nodeFile[n.ID] = n.Attrs["source_file"]
		}
	}

	// Build community membership map
	nodeCommunity := make(map[string]int)
	for cid, members := range communities {
		for _, nodeID := range members {
			nodeCommunity[nodeID] = cid
		}
	}

	// Score each edge
	var candidates []Surprise
	for _, e := range edges {
		// Skip structural edges
		if isStructuralEdge(e.Type) {
			continue
		}

		// Skip edges involving file hubs
		fromNode := nodeMap[e.From]
		toNode := nodeMap[e.To]
		if fromNode == nil || toNode == nil {
			continue
		}
		if isFileHub(fromNode) || isFileHub(toNode) {
			continue
		}

		score, why := surpriseScore(e, nodeFile, nodeCommunity)
		if score <= 0 {
			continue
		}

		fromLabel := fromNode.Label
		if fromLabel == "" {
			fromLabel = e.From
		}
		toLabel := toNode.Label
		if toLabel == "" {
			toLabel = e.To
		}

		candidates = append(candidates, Surprise{
			From:            e.From,
			FromLabel:       fromLabel,
			To:              e.To,
			ToLabel:         toLabel,
			Type:            e.Type,
			Confidence:      string(e.Confidence),
			Score:           score,
			ConfidenceScore: e.ConfidenceScore,
			Why:             why,
			SourceFiles:     []string{nodeFile[e.From], nodeFile[e.To]},
		})
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Take top N
	if topN > len(candidates) {
		topN = len(candidates)
	}

	return candidates[:topN]
}

// surpriseScore calculates how surprising an edge is.
// Returns (score, reason).
func surpriseScore(e *graph.Edge, nodeFile map[string]string, nodeCommunity map[string]int) (float64, string) {
	score := 0.0
	reasons := []string{}

	// 1. Confidence weight - uncertain connections are more noteworthy
	switch e.Confidence {
	case graph.ConfidenceAmbiguous:
		score += 3.0
		reasons = append(reasons, "ambiguous connection - needs verification")
	case graph.ConfidenceInferred:
		score += 2.0
		reasons = append(reasons, "inferred by LLM - not explicit in code")
	case graph.ConfidenceExtracted:
		score += 1.0
	default:
		score += 1.0
	}

	// 2. Cross-file bonus (high weight - cross-file edges are architecturally significant)
	fromFile := nodeFile[e.From]
	toFile := nodeFile[e.To]
	if fromFile != "" && toFile != "" && fromFile != toFile {
		score += 2.5
		reasons = append(reasons, "crosses file boundaries")
	}

	// 3. Cross-community bonus
	fromCom, hasFrom := nodeCommunity[e.From]
	toCom, hasTo := nodeCommunity[e.To]
	if hasFrom && hasTo && fromCom != toCom {
		score += 2.0
		reasons = append(reasons, "bridges different communities")
	}

	// 4. Low confidence score (for INFERRED edges)
	if e.Confidence == graph.ConfidenceInferred && e.ConfidenceScore > 0 && e.ConfidenceScore < 0.7 {
		score += 1.0
		reasons = append(reasons, "low confidence score")
	}

	// 5. Code-doc edge bonus (edges connecting code to documentation/rationale)
	if isCodeDocEdge(e.Type) {
		score += 1.5
		reasons = append(reasons, "code-doc relationship")
	}

	// Build why string
	why := ""
	if len(reasons) > 0 {
		why = reasons[0]
		for i := 1; i < len(reasons); i++ {
			why += "; " + reasons[i]
		}
	}

	return score, why
}

// isStructuralEdge returns true for edges that are mechanical/structural
// rather than semantically interesting.
func isStructuralEdge(edgeType string) bool {
	switch edgeType {
	case graph.EdgeTypeContains, graph.EdgeTypeImports:
		return true
	}
	return false
}

// isCodeDocEdge returns true for edges that connect code to documentation/rationale.
// These edges are particularly valuable as they capture design intent.
func isCodeDocEdge(edgeType string) bool {
	switch edgeType {
	case "rationale_for", "documents", "describes", "explains":
		return true
	}
	return false
}

// AmbiguousEdges returns all edges with AMBIGUOUS confidence.
// These are candidates for human review.
func AmbiguousEdges(edges []*graph.Edge) []*graph.Edge {
	var ambiguous []*graph.Edge
	for _, e := range edges {
		if e.Confidence == graph.ConfidenceAmbiguous {
			ambiguous = append(ambiguous, e)
		}
	}
	return ambiguous
}

// LowConfidenceEdges returns INFERRED edges with confidence score below threshold.
func LowConfidenceEdges(edges []*graph.Edge, threshold float64) []*graph.Edge {
	var lowConf []*graph.Edge
	for _, e := range edges {
		if e.Confidence == graph.ConfidenceInferred && e.ConfidenceScore > 0 && e.ConfidenceScore < threshold {
			lowConf = append(lowConf, e)
		}
	}
	return lowConf
}

// BridgeEdges finds edges that connect otherwise disconnected parts of the graph.
// These are edges whose removal would increase the number of connected components.
// For simplicity, we approximate this by finding cross-community edges.
func BridgeEdges(edges []*graph.Edge, communities map[int][]string) []*graph.Edge {
	// Build community membership map
	nodeCommunity := make(map[string]int)
	for cid, members := range communities {
		for _, nodeID := range members {
			nodeCommunity[nodeID] = cid
		}
	}

	var bridges []*graph.Edge
	for _, e := range edges {
		fromCom, hasFrom := nodeCommunity[e.From]
		toCom, hasTo := nodeCommunity[e.To]
		if hasFrom && hasTo && fromCom != toCom {
			bridges = append(bridges, e)
		}
	}

	return bridges
}
