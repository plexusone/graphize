package analyze

import (
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestCalculateBetweenness(t *testing.T) {
	// Create a simple graph where node B is a bridge between A-B-C
	//   A -- B -- C
	nodes := []*graph.Node{
		{ID: "A", Type: "function", Label: "funcA"},
		{ID: "B", Type: "function", Label: "funcB"},
		{ID: "C", Type: "function", Label: "funcC"},
	}

	edges := []*graph.Edge{
		{From: "A", To: "B", Type: "calls"},
		{From: "B", To: "C", Type: "calls"},
	}

	opts := BetweennessOptions{
		TopN: 10,
	}

	result := CalculateBetweenness(nodes, edges, opts)

	// B should have highest betweenness (it's the bridge)
	if result.Scores["B"] <= result.Scores["A"] {
		t.Errorf("B should have higher betweenness than A: B=%f, A=%f",
			result.Scores["B"], result.Scores["A"])
	}
	if result.Scores["B"] <= result.Scores["C"] {
		t.Errorf("B should have higher betweenness than C: B=%f, C=%f",
			result.Scores["B"], result.Scores["C"])
	}

	// Should have bridges
	if len(result.Bridges) == 0 {
		t.Error("Should have at least one bridge")
	}

	// First bridge should be B
	if len(result.Bridges) > 0 && result.Bridges[0].ID != "B" {
		t.Errorf("First bridge should be B, got %s", result.Bridges[0].ID)
	}
}

func TestCalculateBetweennessStarGraph(t *testing.T) {
	// Star graph: center node connected to all others
	//     A
	//     |
	// B - C - D
	//     |
	//     E
	nodes := []*graph.Node{
		{ID: "center", Type: "function", Label: "center"},
		{ID: "A", Type: "function", Label: "A"},
		{ID: "B", Type: "function", Label: "B"},
		{ID: "D", Type: "function", Label: "D"},
		{ID: "E", Type: "function", Label: "E"},
	}

	edges := []*graph.Edge{
		{From: "center", To: "A", Type: "calls"},
		{From: "center", To: "B", Type: "calls"},
		{From: "center", To: "D", Type: "calls"},
		{From: "center", To: "E", Type: "calls"},
	}

	result := CalculateBetweenness(nodes, edges, DefaultBetweennessOptions())

	// Center should have highest betweenness
	for _, nodeID := range []string{"A", "B", "D", "E"} {
		if result.Scores["center"] < result.Scores[nodeID] {
			t.Errorf("center should have higher betweenness than %s: center=%f, %s=%f",
				nodeID, result.Scores["center"], nodeID, result.Scores[nodeID])
		}
	}
}

func TestCalculateBetweennessExcludeTypes(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "pkg", Type: "package", Label: "mypkg"},
		{ID: "A", Type: "function", Label: "funcA"},
		{ID: "B", Type: "function", Label: "funcB"},
	}

	edges := []*graph.Edge{
		{From: "pkg", To: "A", Type: "contains"},
		{From: "pkg", To: "B", Type: "contains"},
		{From: "A", To: "B", Type: "calls"},
	}

	opts := BetweennessOptions{
		TopN:             10,
		ExcludeNodeTypes: []string{"package"},
		ExcludeEdgeTypes: []string{"contains"},
	}

	result := CalculateBetweenness(nodes, edges, opts)

	// Package should not be in bridges (excluded)
	for _, bridge := range result.Bridges {
		if bridge.Type == "package" {
			t.Error("Package nodes should be excluded from bridges")
		}
	}
}

func TestCalculateBetweennessWithCommunities(t *testing.T) {
	// Two communities connected by a bridge node
	// Community 1: A, B, Bridge
	// Community 2: Bridge, C, D
	nodes := []*graph.Node{
		{ID: "A", Type: "function", Label: "A"},
		{ID: "B", Type: "function", Label: "B"},
		{ID: "Bridge", Type: "function", Label: "Bridge"},
		{ID: "C", Type: "function", Label: "C"},
		{ID: "D", Type: "function", Label: "D"},
	}

	edges := []*graph.Edge{
		// Community 1 internal
		{From: "A", To: "B", Type: "calls"},
		{From: "A", To: "Bridge", Type: "calls"},
		{From: "B", To: "Bridge", Type: "calls"},
		// Community 2 internal
		{From: "C", To: "D", Type: "calls"},
		{From: "C", To: "Bridge", Type: "calls"},
		{From: "D", To: "Bridge", Type: "calls"},
	}

	communities := map[int][]string{
		1: {"A", "B", "Bridge"},
		2: {"Bridge", "C", "D"},
	}

	result := FindBridgesWithCommunities(nodes, edges, communities, 10)

	// Bridge should be first and connect both communities
	if len(result) == 0 {
		t.Fatal("Should have bridges")
	}

	found := false
	for _, bridge := range result {
		if bridge.ID == "Bridge" {
			found = true
			if len(bridge.ConnectsCommunities) < 2 {
				t.Errorf("Bridge should connect multiple communities, got %v",
					bridge.ConnectsCommunities)
			}
			break
		}
	}
	if !found {
		t.Error("Bridge node should be in results")
	}
}

func TestFindBridges(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "A", Type: "function", Label: "A"},
		{ID: "B", Type: "function", Label: "B"},
		{ID: "C", Type: "function", Label: "C"},
	}

	edges := []*graph.Edge{
		{From: "A", To: "B", Type: "calls"},
		{From: "B", To: "C", Type: "calls"},
	}

	bridges := FindBridges(nodes, edges, 5)

	if len(bridges) == 0 {
		t.Error("Should find at least one bridge")
	}
}

func TestCalculateBetweennessEmptyGraph(t *testing.T) {
	result := CalculateBetweenness(nil, nil, DefaultBetweennessOptions())

	if len(result.Bridges) != 0 {
		t.Error("Empty graph should have no bridges")
	}
	if len(result.Scores) != 0 {
		t.Error("Empty graph should have no scores")
	}
}

func TestDefaultBetweennessOptions(t *testing.T) {
	opts := DefaultBetweennessOptions()

	if opts.TopN != 20 {
		t.Errorf("Default TopN should be 20, got %d", opts.TopN)
	}

	// Should exclude package and file by default
	excludeTypes := make(map[string]bool)
	for _, t := range opts.ExcludeNodeTypes {
		excludeTypes[t] = true
	}
	if !excludeTypes["package"] {
		t.Error("Should exclude package nodes by default")
	}
	if !excludeTypes["file"] {
		t.Error("Should exclude file nodes by default")
	}
}
