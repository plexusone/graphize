package analyze

import (
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestGroupEdgesByType(t *testing.T) {
	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls"},
		{From: "b", To: "c", Type: "calls"},
		{From: "a", To: "c", Type: "imports"},
	}

	result := GroupEdgesByType(edges)

	if len(result["calls"]) != 2 {
		t.Errorf("expected 2 'calls' edges, got %d", len(result["calls"]))
	}
	if len(result["imports"]) != 1 {
		t.Errorf("expected 1 'imports' edge, got %d", len(result["imports"]))
	}
}

func TestCountEdgesByType(t *testing.T) {
	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls"},
		{From: "b", To: "c", Type: "calls"},
		{From: "a", To: "c", Type: "imports"},
		{From: "c", To: "d", Type: "contains"},
	}

	result := CountEdgesByType(edges)

	if result["calls"] != 2 {
		t.Errorf("expected 2 'calls', got %d", result["calls"])
	}
	if result["imports"] != 1 {
		t.Errorf("expected 1 'imports', got %d", result["imports"])
	}
	if result["contains"] != 1 {
		t.Errorf("expected 1 'contains', got %d", result["contains"])
	}
}

func TestGroupEdgesByConfidence(t *testing.T) {
	edges := []*graph.Edge{
		{From: "a", To: "b", Confidence: graph.ConfidenceExtracted},
		{From: "b", To: "c", Confidence: graph.ConfidenceExtracted},
		{From: "a", To: "c", Confidence: graph.ConfidenceInferred},
	}

	result := GroupEdgesByConfidence(edges)

	if len(result[string(graph.ConfidenceExtracted)]) != 2 {
		t.Errorf("expected 2 EXTRACTED edges, got %d", len(result[string(graph.ConfidenceExtracted)]))
	}
	if len(result[string(graph.ConfidenceInferred)]) != 1 {
		t.Errorf("expected 1 INFERRED edge, got %d", len(result[string(graph.ConfidenceInferred)]))
	}
}

func TestCountEdgesByConfidence(t *testing.T) {
	edges := []*graph.Edge{
		{From: "a", To: "b", Confidence: graph.ConfidenceExtracted},
		{From: "b", To: "c", Confidence: graph.ConfidenceExtracted},
		{From: "a", To: "c", Confidence: graph.ConfidenceInferred},
		{From: "c", To: "d", Confidence: graph.ConfidenceAmbiguous},
	}

	result := CountEdgesByConfidence(edges)

	if result[string(graph.ConfidenceExtracted)] != 2 {
		t.Errorf("expected 2 EXTRACTED, got %d", result[string(graph.ConfidenceExtracted)])
	}
	if result[string(graph.ConfidenceInferred)] != 1 {
		t.Errorf("expected 1 INFERRED, got %d", result[string(graph.ConfidenceInferred)])
	}
	if result[string(graph.ConfidenceAmbiguous)] != 1 {
		t.Errorf("expected 1 AMBIGUOUS, got %d", result[string(graph.ConfidenceAmbiguous)])
	}
}
