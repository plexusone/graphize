package query

import (
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestFilterEdges(t *testing.T) {
	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls"},
		{From: "b", To: "c", Type: "calls"},
		{From: "a", To: "c", Type: "imports"},
		{From: "d", To: "a", Type: "references"},
	}

	tests := []struct {
		name     string
		filter   EdgeFilter
		expected int
	}{
		{
			name:     "no filter",
			filter:   EdgeFilter{},
			expected: 4,
		},
		{
			name:     "filter by node ID",
			filter:   EdgeFilter{NodeID: "a"},
			expected: 3, // a->b, a->c, d->a
		},
		{
			name:     "filter by from",
			filter:   EdgeFilter{From: "a"},
			expected: 2, // a->b, a->c
		},
		{
			name:     "filter by type",
			filter:   EdgeFilter{Type: "calls"},
			expected: 2, // a->b, b->c
		},
		{
			name:     "filter by multiple types",
			filter:   EdgeFilter{Types: []string{"calls", "imports"}},
			expected: 3, // a->b, b->c, a->c
		},
		{
			name:     "filter with limit",
			filter:   EdgeFilter{Limit: 2},
			expected: 2,
		},
		{
			name:     "combined filters",
			filter:   EdgeFilter{NodeID: "a", Type: "calls"},
			expected: 1, // only a->b
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterEdges(edges, tt.filter)
			if result.Matches != tt.expected {
				t.Errorf("expected %d matches, got %d", tt.expected, result.Matches)
			}
		})
	}
}

func TestFilterEdgesTruncation(t *testing.T) {
	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls"},
		{From: "b", To: "c", Type: "calls"},
		{From: "c", To: "d", Type: "calls"},
	}

	result := FilterEdges(edges, EdgeFilter{Limit: 2})

	if !result.Truncated {
		t.Error("expected truncated=true")
	}
	if len(result.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(result.Edges))
	}
}

func TestFindPartialMatches(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"func_main":     {ID: "func_main"},
			"func_helper":   {ID: "func_helper"},
			"func_process":  {ID: "func_process"},
			"type_User":     {ID: "type_User"},
			"pkg_mypackage": {ID: "pkg_mypackage"},
		},
	}

	result := FindPartialMatches(g, "func", 10)

	if len(result.Matches) != 3 {
		t.Errorf("expected 3 matches for 'func', got %d", len(result.Matches))
	}
}

func TestFindPartialMatchesWithLimit(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"func_a": {ID: "func_a"},
			"func_b": {ID: "func_b"},
			"func_c": {ID: "func_c"},
		},
	}

	result := FindPartialMatches(g, "func", 2)

	if len(result.Matches) != 2 {
		t.Errorf("expected 2 matches (limited), got %d", len(result.Matches))
	}
}
