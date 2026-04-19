package query

import (
	"github.com/plexusone/graphfs/pkg/graph"
)

// EdgeFilter defines criteria for filtering edges.
type EdgeFilter struct {
	// NodeID filters edges that touch this node (from or to).
	NodeID string

	// From filters edges by source node.
	From string

	// To filters edges by target node.
	To string

	// Type filters edges by edge type.
	Type string

	// Types filters edges by multiple edge types (OR).
	Types []string

	// Limit is the maximum number of edges to return.
	Limit int
}

// EdgeListOutput represents filtered edge results.
type EdgeListOutput struct {
	Query     string               `json:"query,omitempty"`
	Matches   int                  `json:"matches"`
	Edges     []EdgeOutputWithMeta `json:"edges"`
	Truncated bool                 `json:"truncated,omitempty"`
	Message   string               `json:"message,omitempty"`
}

// EdgeOutputWithMeta includes confidence metadata.
type EdgeOutputWithMeta struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	Confidence string `json:"confidence,omitempty"`
}

// FilterEdges filters a list of edges according to the filter criteria.
func FilterEdges(edges []*graph.Edge, filter EdgeFilter) *EdgeListOutput {
	var matches []EdgeOutputWithMeta

	typeSet := make(map[string]bool)
	for _, t := range filter.Types {
		typeSet[t] = true
	}
	if filter.Type != "" {
		typeSet[filter.Type] = true
	}

	for _, e := range edges {
		// Filter by node ID (touches either end)
		if filter.NodeID != "" && e.From != filter.NodeID && e.To != filter.NodeID {
			continue
		}

		// Filter by from
		if filter.From != "" && e.From != filter.From {
			continue
		}

		// Filter by to
		if filter.To != "" && e.To != filter.To {
			continue
		}

		// Filter by type(s)
		if len(typeSet) > 0 && !typeSet[e.Type] {
			continue
		}

		matches = append(matches, EdgeOutputWithMeta{
			From:       e.From,
			To:         e.To,
			Type:       e.Type,
			Confidence: string(e.Confidence),
		})

		if filter.Limit > 0 && len(matches) >= filter.Limit {
			break
		}
	}

	output := &EdgeListOutput{
		Query:   filter.NodeID,
		Matches: len(matches),
		Edges:   matches,
	}

	if filter.Limit > 0 && len(matches) >= filter.Limit {
		output.Truncated = true
		output.Message = "Results truncated. Use --limit to increase."
	}

	return output
}

// NodeMatch represents a partial node match suggestion.
type NodeMatch struct {
	Query   string   `json:"query"`
	Matches []string `json:"matches"`
	Message string   `json:"message"`
}

// FindPartialMatches finds nodes that partially match the query.
func FindPartialMatches(g *graph.Graph, query string, limit int) *NodeMatch {
	var matches []string
	for id := range g.Nodes {
		if containsIgnoreCase(id, query) {
			matches = append(matches, id)
		}
	}

	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return &NodeMatch{
		Query:   query,
		Matches: matches,
		Message: "Did you mean one of these?",
	}
}

func containsIgnoreCase(s, substr string) bool {
	// Simple contains for now
	return len(substr) > 0 && len(s) >= len(substr) && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
