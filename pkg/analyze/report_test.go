package analyze

import (
	"strings"
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestGenerateReport(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "pkg_main", Type: "package", Label: "main"},
		{ID: "func_main", Type: "function", Label: "main", Attrs: map[string]string{"package": "main", "source_file": "main.go"}},
		{ID: "func_helper", Type: "function", Label: "helper", Attrs: map[string]string{"package": "main", "source_file": "helper.go"}},
		{ID: "type_Config", Type: "struct", Label: "Config", Attrs: map[string]string{"package": "main", "source_file": "config.go"}},
	}

	edges := []*graph.Edge{
		{From: "pkg_main", To: "func_main", Type: "contains"},
		{From: "pkg_main", To: "func_helper", Type: "contains"},
		{From: "pkg_main", To: "type_Config", Type: "contains"},
		{From: "func_main", To: "func_helper", Type: "calls", Confidence: graph.ConfidenceExtracted},
		{From: "func_main", To: "type_Config", Type: "references", Confidence: graph.ConfidenceInferred},
	}

	opts := DefaultReportOptions()
	report := GenerateReport(nodes, edges, opts)

	// Verify summary
	if report.Summary.TotalNodes != 4 {
		t.Errorf("expected 4 nodes, got %d", report.Summary.TotalNodes)
	}
	if report.Summary.TotalEdges != 5 {
		t.Errorf("expected 5 edges, got %d", report.Summary.TotalEdges)
	}

	// Verify node types counted
	if report.Summary.NodeTypeCounts["function"] != 2 {
		t.Errorf("expected 2 function nodes, got %d", report.Summary.NodeTypeCounts["function"])
	}

	// Verify edge types counted
	if report.Summary.EdgeTypeCounts["contains"] != 3 {
		t.Errorf("expected 3 contains edges, got %d", report.Summary.EdgeTypeCounts["contains"])
	}
}

func TestReportFormatMarkdown(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "pkg_main", Type: "package", Label: "main"},
		{ID: "func_main", Type: "function", Label: "main"},
	}

	edges := []*graph.Edge{
		{From: "pkg_main", To: "func_main", Type: "contains"},
	}

	opts := DefaultReportOptions()
	report := GenerateReport(nodes, edges, opts)
	markdown := report.FormatMarkdown(opts)

	// Verify markdown has expected sections
	expectedSections := []string{
		"# Graph Analysis Report",
		"## Summary",
		"### Node Types",
		"### Edge Types",
		"## God Nodes",
		"## Bridges",
		"## Communities",
		"## Surprising Connections",
		"## Isolated Nodes",
		"## Package Statistics",
		"## Cross-File Dependencies",
		"## Suggested Questions",
	}

	for _, section := range expectedSections {
		if !strings.Contains(markdown, section) {
			t.Errorf("expected markdown to contain section %q", section)
		}
	}
}

func TestReportSummary(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "a", Type: "function"},
		{ID: "b", Type: "function"},
		{ID: "c", Type: "struct"},
	}

	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls", Confidence: graph.ConfidenceExtracted},
		{From: "a", To: "c", Type: "references", Confidence: graph.ConfidenceInferred},
	}

	opts := DefaultReportOptions()
	report := GenerateReport(nodes, edges, opts)

	// Check node type counts
	if report.Summary.NodeTypeCounts["function"] != 2 {
		t.Errorf("expected 2 functions, got %d", report.Summary.NodeTypeCounts["function"])
	}
	if report.Summary.NodeTypeCounts["struct"] != 1 {
		t.Errorf("expected 1 struct, got %d", report.Summary.NodeTypeCounts["struct"])
	}

	// Check edge type counts
	if report.Summary.EdgeTypeCounts["calls"] != 1 {
		t.Errorf("expected 1 calls edge, got %d", report.Summary.EdgeTypeCounts["calls"])
	}

	// Check confidence counts
	if report.Summary.ConfidenceCounts["EXTRACTED"] != 1 {
		t.Errorf("expected 1 EXTRACTED edge, got %d", report.Summary.ConfidenceCounts["EXTRACTED"])
	}
	if report.Summary.ConfidenceCounts["INFERRED"] != 1 {
		t.Errorf("expected 1 INFERRED edge, got %d", report.Summary.ConfidenceCounts["INFERRED"])
	}
}

func TestEmptyReport(t *testing.T) {
	opts := DefaultReportOptions()
	report := GenerateReport([]*graph.Node{}, []*graph.Edge{}, opts)

	if report.Summary.TotalNodes != 0 {
		t.Errorf("expected 0 nodes, got %d", report.Summary.TotalNodes)
	}
	if report.Summary.TotalEdges != 0 {
		t.Errorf("expected 0 edges, got %d", report.Summary.TotalEdges)
	}

	// Markdown should still render without errors
	markdown := report.FormatMarkdown(opts)
	if !strings.Contains(markdown, "# Graph Analysis Report") {
		t.Error("expected markdown header in empty report")
	}
}

func TestReportWithCrossFileEdges(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "a", Type: "function", Attrs: map[string]string{"source_file": "a.go"}},
		{ID: "b", Type: "function", Attrs: map[string]string{"source_file": "b.go"}},
		{ID: "c", Type: "function", Attrs: map[string]string{"source_file": "a.go"}},
	}

	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls"}, // cross-file
		{From: "a", To: "c", Type: "calls"}, // same file
	}

	opts := DefaultReportOptions()
	report := GenerateReport(nodes, edges, opts)

	if report.CrossFileEdgeCount != 1 {
		t.Errorf("expected 1 cross-file edge, got %d", report.CrossFileEdgeCount)
	}
}
