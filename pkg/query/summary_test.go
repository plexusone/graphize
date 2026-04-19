package query

import (
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestComputeSummary(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"func_main":     {ID: "func_main", Type: "function"},
			"func_helper":   {ID: "func_helper", Type: "function"},
			"pkg_mypackage": {ID: "pkg_mypackage", Type: "package"},
			"type_User":     {ID: "type_User", Type: "type"},
		},
		Edges: []*graph.Edge{
			{From: "func_main", To: "func_helper", Type: "calls"},
			{From: "func_main", To: "type_User", Type: "references"},
			{From: "pkg_mypackage", To: "func_main", Type: "contains"},
			{From: "pkg_mypackage", To: "func_helper", Type: "contains"},
		},
	}

	summary := ComputeSummary(g)

	if summary.TotalNodes != 4 {
		t.Errorf("expected 4 nodes, got %d", summary.TotalNodes)
	}

	if summary.TotalEdges != 4 {
		t.Errorf("expected 4 edges, got %d", summary.TotalEdges)
	}

	if summary.NodeTypes["function"] != 2 {
		t.Errorf("expected 2 functions, got %d", summary.NodeTypes["function"])
	}

	if summary.EdgeTypes["calls"] != 1 {
		t.Errorf("expected 1 calls edge, got %d", summary.EdgeTypes["calls"])
	}

	if summary.EdgeTypes["contains"] != 2 {
		t.Errorf("expected 2 contains edges, got %d", summary.EdgeTypes["contains"])
	}

	// func_main has 3 edges (calls helper, references User, contained by package)
	// pkg_mypackage has 2 edges (contains main, contains helper)
	if len(summary.GodNodes) == 0 {
		t.Error("expected at least one god node")
	}
}

func TestComputeSummaryWithOptions(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"func_main":   {ID: "func_main", Type: "function"},
			"call_extern": {ID: "call_extern", Type: "call"},
		},
		Edges: []*graph.Edge{
			{From: "func_main", To: "call_extern", Type: "calls"},
			{From: "call_extern", To: "func_main", Type: "returns"},
		},
	}

	opts := SummaryOptions{
		MaxGodNodes:     5,
		ExcludePrefixes: []string{"call_"},
	}

	summary := ComputeSummaryWithOptions(g, opts)

	// call_extern should be excluded from god nodes
	for _, god := range summary.GodNodes {
		if god.ID == "call_extern" {
			t.Error("call_extern should be excluded from god nodes")
		}
	}
}

func TestComputeSummaryFromLists(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "a", Type: "function"},
		{ID: "b", Type: "function"},
	}
	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls"},
	}

	summary := ComputeSummaryFromLists(nodes, edges)

	if summary.TotalNodes != 2 {
		t.Errorf("expected 2 nodes, got %d", summary.TotalNodes)
	}
	if summary.TotalEdges != 1 {
		t.Errorf("expected 1 edge, got %d", summary.TotalEdges)
	}
}
