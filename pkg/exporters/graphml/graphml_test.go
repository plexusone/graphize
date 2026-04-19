package graphml

import (
	"bytes"
	"strings"
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()

	if !g.Directed {
		t.Error("expected Directed=true by default")
	}
	if g.GraphID != "code-graph" {
		t.Errorf("expected GraphID='code-graph', got %q", g.GraphID)
	}
	if g.Description != "graphize export" {
		t.Errorf("expected Description='graphize export', got %q", g.Description)
	}
}

func TestGenerate(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
		{ID: "func_helper", Type: "function", Label: "helper"},
	}

	edges := []*graph.Edge{
		{From: "func_main", To: "func_helper", Type: "calls", Confidence: graph.ConfidenceExtracted},
	}

	g := NewGenerator()
	result, err := g.Generate(nodes, edges)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.NodeCount != 2 {
		t.Errorf("expected 2 nodes, got %d", result.NodeCount)
	}
	if result.EdgeCount != 1 {
		t.Errorf("expected 1 edge, got %d", result.EdgeCount)
	}
	if len(result.Data) == 0 {
		t.Error("expected non-empty output")
	}

	// Verify GraphML structure
	output := string(result.Data)
	if !strings.Contains(output, "<graphml") {
		t.Error("expected graphml element")
	}
	if !strings.Contains(output, "func_main") {
		t.Error("expected func_main node ID in output")
	}
	if !strings.Contains(output, "func_helper") {
		t.Error("expected func_helper node ID in output")
	}
}

func TestGenerateWithAttributes(t *testing.T) {
	nodes := []*graph.Node{
		{
			ID:    "func_main",
			Type:  "function",
			Label: "main",
			Attrs: map[string]string{
				"package":     "main",
				"source_file": "main.go",
			},
		},
	}

	edges := []*graph.Edge{}

	g := NewGenerator()
	result, err := g.Generate(nodes, edges)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(result.Data)
	if !strings.Contains(output, "main.go") {
		t.Error("expected source_file attribute in output")
	}
}

func TestGenerateUndirected(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "a", Type: "function", Label: "a"},
		{ID: "b", Type: "function", Label: "b"},
	}

	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls"},
	}

	g := NewGenerator()
	g.Directed = false

	result, err := g.Generate(nodes, edges)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(result.Data)
	if !strings.Contains(output, `edgedefault="undirected"`) {
		t.Error("expected undirected edge default")
	}
}

func TestGenerateWithConfidenceScore(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "a", Type: "function", Label: "a"},
		{ID: "b", Type: "function", Label: "b"},
	}

	edges := []*graph.Edge{
		{
			From:            "a",
			To:              "b",
			Type:            "inferred_depends",
			Confidence:      graph.ConfidenceInferred,
			ConfidenceScore: 0.85,
		},
	}

	g := NewGenerator()
	result, err := g.Generate(nodes, edges)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(result.Data)
	if !strings.Contains(output, "INFERRED") {
		t.Error("expected INFERRED confidence in output")
	}
	if !strings.Contains(output, "0.85") {
		t.Error("expected confidence score in output")
	}
}

func TestGenerateSkippedEdges(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "a", Type: "function", Label: "a"},
	}

	// Edge references missing node
	edges := []*graph.Edge{
		{From: "a", To: "missing", Type: "calls"},
	}

	g := NewGenerator()
	result, err := g.Generate(nodes, edges)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.SkippedEdges != 1 {
		t.Errorf("expected 1 skipped edge, got %d", result.SkippedEdges)
	}
	if result.EdgeCount != 0 {
		t.Errorf("expected 0 edges, got %d", result.EdgeCount)
	}
}

func TestWriteTo(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "a", Type: "function", Label: "a"},
	}
	edges := []*graph.Edge{}

	var buf bytes.Buffer
	g := NewGenerator()

	result, err := g.WriteTo(&buf, nodes, edges)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if result.NodeCount != 1 {
		t.Errorf("expected 1 node, got %d", result.NodeCount)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty buffer")
	}
}

func TestGenerateEmpty(t *testing.T) {
	g := NewGenerator()
	result, err := g.Generate([]*graph.Node{}, []*graph.Edge{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.NodeCount != 0 {
		t.Errorf("expected 0 nodes, got %d", result.NodeCount)
	}
	if result.EdgeCount != 0 {
		t.Errorf("expected 0 edges, got %d", result.EdgeCount)
	}

	// Should still produce valid GraphML
	output := string(result.Data)
	if !strings.Contains(output, "<graphml") {
		t.Error("expected graphml element in empty graph")
	}
}

func TestCustomGraphID(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "a", Type: "function", Label: "a"},
	}
	edges := []*graph.Edge{}

	g := NewGenerator()
	g.GraphID = "my-custom-graph"

	result, err := g.Generate(nodes, edges)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(result.Data)
	if !strings.Contains(output, "my-custom-graph") {
		t.Error("expected custom graph ID in output")
	}
}
