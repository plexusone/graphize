package extract

import (
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestChunkFiles(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		chunkSize int
		wantLen   int
		wantSizes []int
	}{
		{
			name:      "empty slice",
			files:     []string{},
			chunkSize: 5,
			wantLen:   0,
			wantSizes: []int{},
		},
		{
			name:      "fewer files than chunk size",
			files:     []string{"a.go", "b.go", "c.go"},
			chunkSize: 5,
			wantLen:   1,
			wantSizes: []int{3},
		},
		{
			name:      "exact chunk size",
			files:     []string{"a.go", "b.go", "c.go", "d.go", "e.go"},
			chunkSize: 5,
			wantLen:   1,
			wantSizes: []int{5},
		},
		{
			name:      "multiple full chunks",
			files:     []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go"},
			chunkSize: 3,
			wantLen:   2,
			wantSizes: []int{3, 3},
		},
		{
			name:      "partial last chunk",
			files:     []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go", "g.go"},
			chunkSize: 3,
			wantLen:   3,
			wantSizes: []int{3, 3, 1},
		},
		{
			name:      "chunk size of 1",
			files:     []string{"a.go", "b.go", "c.go"},
			chunkSize: 1,
			wantLen:   3,
			wantSizes: []int{1, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkFiles(tt.files, tt.chunkSize)

			if len(chunks) != tt.wantLen {
				t.Errorf("ChunkFiles() returned %d chunks, want %d", len(chunks), tt.wantLen)
			}

			for i, chunk := range chunks {
				if i < len(tt.wantSizes) && len(chunk) != tt.wantSizes[i] {
					t.Errorf("chunk[%d] has %d files, want %d", i, len(chunk), tt.wantSizes[i])
				}
			}
		})
	}
}

func TestParseSemanticJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNodes int
		wantEdges int
		wantErr   bool
	}{
		{
			name: "valid JSON",
			input: `{
				"nodes": [],
				"edges": [
					{
						"from": "func_a",
						"to": "func_b",
						"type": "similar_to",
						"confidence": "INFERRED",
						"confidence_score": 0.8,
						"reason": "test"
					}
				]
			}`,
			wantNodes: 0,
			wantEdges: 1,
			wantErr:   false,
		},
		{
			name: "JSON in markdown code block",
			input: "```json\n{\"nodes\": [], \"edges\": []}\n```",
			wantNodes: 0,
			wantEdges: 0,
			wantErr:   false,
		},
		{
			name: "JSON in plain code block",
			input: "```\n{\"nodes\": [], \"edges\": []}\n```",
			wantNodes: 0,
			wantEdges: 0,
			wantErr:   false,
		},
		{
			name:      "invalid JSON",
			input:     `{"nodes": [}`,
			wantNodes: 0,
			wantEdges: 0,
			wantErr:   true,
		},
		{
			name: "multiple edges",
			input: `{
				"nodes": [{"id": "type_Foo", "type": "struct", "label": "Foo"}],
				"edges": [
					{"from": "a", "to": "b", "type": "similar_to", "confidence": "INFERRED", "confidence_score": 0.9, "reason": "r1"},
					{"from": "c", "to": "d", "type": "shared_concern", "confidence": "AMBIGUOUS", "confidence_score": 0.2, "reason": "r2"}
				]
			}`,
			wantNodes: 1,
			wantEdges: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSemanticJSON([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSemanticJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(result.Nodes) != tt.wantNodes {
					t.Errorf("ParseSemanticJSON() got %d nodes, want %d", len(result.Nodes), tt.wantNodes)
				}
				if len(result.Edges) != tt.wantEdges {
					t.Errorf("ParseSemanticJSON() got %d edges, want %d", len(result.Edges), tt.wantEdges)
				}
			}
		})
	}
}

func TestValidateSemanticExtraction(t *testing.T) {
	tests := []struct {
		name    string
		ext     *SemanticExtraction
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid extraction",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "a", To: "b", Type: "similar_to", Confidence: "INFERRED", ConfidenceScore: 0.8},
				},
			},
			wantErr: false,
		},
		{
			name: "valid AMBIGUOUS confidence",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "a", To: "b", Type: "similar_to", Confidence: "AMBIGUOUS", ConfidenceScore: 0.2},
				},
			},
			wantErr: false,
		},
		{
			name: "missing from field",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "", To: "b", Type: "similar_to", Confidence: "INFERRED", ConfidenceScore: 0.8},
				},
			},
			wantErr: true,
			errMsg:  "missing 'from'",
		},
		{
			name: "missing to field",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "a", To: "", Type: "similar_to", Confidence: "INFERRED", ConfidenceScore: 0.8},
				},
			},
			wantErr: true,
			errMsg:  "missing 'to'",
		},
		{
			name: "missing type field",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "a", To: "b", Type: "", Confidence: "INFERRED", ConfidenceScore: 0.8},
				},
			},
			wantErr: true,
			errMsg:  "missing 'type'",
		},
		{
			name: "EXTRACTED confidence not allowed",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "a", To: "b", Type: "similar_to", Confidence: "EXTRACTED", ConfidenceScore: 1.0},
				},
			},
			wantErr: true,
			errMsg:  "EXTRACTED",
		},
		{
			name: "invalid confidence value",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "a", To: "b", Type: "similar_to", Confidence: "UNKNOWN", ConfidenceScore: 0.5},
				},
			},
			wantErr: true,
			errMsg:  "invalid confidence",
		},
		{
			name: "confidence score too low",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "a", To: "b", Type: "similar_to", Confidence: "INFERRED", ConfidenceScore: -0.1},
				},
			},
			wantErr: true,
			errMsg:  "confidence_score",
		},
		{
			name: "confidence score too high",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "a", To: "b", Type: "similar_to", Confidence: "INFERRED", ConfidenceScore: 1.5},
				},
			},
			wantErr: true,
			errMsg:  "confidence_score",
		},
		{
			name: "empty edges is valid",
			ext: &SemanticExtraction{
				Edges: []SemanticEdge{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSemanticExtraction(tt.ext)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSemanticExtraction() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateSemanticExtraction() error = %v, should contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMergeExtractions(t *testing.T) {
	tests := []struct {
		name           string
		astNodes       []*graph.Node
		astEdges       []*graph.Edge
		semantic       *SemanticExtraction
		wantNodes      int
		wantEdges      int
		wantNewNodes   int
		wantNewEdges   int
	}{
		{
			name: "merge with no duplicates",
			astNodes: []*graph.Node{
				{ID: "func_a", Type: "function", Label: "a"},
				{ID: "func_b", Type: "function", Label: "b"},
			},
			astEdges: []*graph.Edge{
				{From: "func_a", To: "func_b", Type: "calls"},
			},
			semantic: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "func_a", To: "func_b", Type: "similar_to", Confidence: "INFERRED", ConfidenceScore: 0.8, Reason: "test"},
				},
			},
			wantNodes:    2,
			wantEdges:    2,
			wantNewNodes: 0,
			wantNewEdges: 1,
		},
		{
			name: "skip duplicate edges",
			astNodes: []*graph.Node{
				{ID: "func_a", Type: "function", Label: "a"},
			},
			astEdges: []*graph.Edge{
				{From: "func_a", To: "func_b", Type: "calls"},
			},
			semantic: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "func_a", To: "func_b", Type: "calls", Confidence: "INFERRED", ConfidenceScore: 0.8, Reason: "test"},
				},
			},
			wantNodes:    1,
			wantEdges:    1,
			wantNewNodes: 0,
			wantNewEdges: 0, // Duplicate should be skipped
		},
		{
			name: "add new nodes",
			astNodes: []*graph.Node{
				{ID: "func_a", Type: "function", Label: "a"},
			},
			astEdges: []*graph.Edge{},
			semantic: &SemanticExtraction{
				Nodes: []SemanticNode{
					{ID: "type_Foo", Type: "struct", Label: "Foo"},
				},
				Edges: []SemanticEdge{},
			},
			wantNodes:    2,
			wantEdges:    0,
			wantNewNodes: 1,
			wantNewEdges: 0,
		},
		{
			name: "skip duplicate nodes",
			astNodes: []*graph.Node{
				{ID: "func_a", Type: "function", Label: "a"},
			},
			astEdges: []*graph.Edge{},
			semantic: &SemanticExtraction{
				Nodes: []SemanticNode{
					{ID: "func_a", Type: "function", Label: "a_updated"}, // Same ID
				},
				Edges: []SemanticEdge{},
			},
			wantNodes:    1, // Should not add duplicate
			wantEdges:    0,
			wantNewNodes: 0,
			wantNewEdges: 0,
		},
		{
			name:     "empty AST graph",
			astNodes: []*graph.Node{},
			astEdges: []*graph.Edge{},
			semantic: &SemanticExtraction{
				Nodes: []SemanticNode{
					{ID: "type_Foo", Type: "struct", Label: "Foo"},
				},
				Edges: []SemanticEdge{
					{From: "type_Foo", To: "type_Bar", Type: "similar_to", Confidence: "INFERRED", ConfidenceScore: 0.7, Reason: "test"},
				},
			},
			wantNodes:    1,
			wantEdges:    1,
			wantNewNodes: 1,
			wantNewEdges: 1,
		},
		{
			name: "AMBIGUOUS confidence mapping",
			astNodes: []*graph.Node{
				{ID: "func_a", Type: "function", Label: "a"},
			},
			astEdges: []*graph.Edge{},
			semantic: &SemanticExtraction{
				Edges: []SemanticEdge{
					{From: "func_a", To: "func_b", Type: "similar_to", Confidence: "AMBIGUOUS", ConfidenceScore: 0.2, Reason: "uncertain"},
				},
			},
			wantNodes:    1,
			wantEdges:    1,
			wantNewNodes: 0,
			wantNewEdges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergedNodes, mergedEdges := MergeExtractions(tt.astNodes, tt.astEdges, tt.semantic)

			if len(mergedNodes) != tt.wantNodes {
				t.Errorf("MergeExtractions() got %d nodes, want %d", len(mergedNodes), tt.wantNodes)
			}

			if len(mergedEdges) != tt.wantEdges {
				t.Errorf("MergeExtractions() got %d edges, want %d", len(mergedEdges), tt.wantEdges)
			}

			newNodes := len(mergedNodes) - len(tt.astNodes)
			if newNodes != tt.wantNewNodes {
				t.Errorf("MergeExtractions() added %d new nodes, want %d", newNodes, tt.wantNewNodes)
			}

			newEdges := len(mergedEdges) - len(tt.astEdges)
			if newEdges != tt.wantNewEdges {
				t.Errorf("MergeExtractions() added %d new edges, want %d", newEdges, tt.wantNewEdges)
			}
		})
	}
}

func TestIsValidSemanticEdgeType(t *testing.T) {
	validTypes := []string{
		"inferred_depends",
		"rationale_for",
		"similar_to",
		"implements_pattern",
		"shared_concern",
	}

	invalidTypes := []string{
		"calls",
		"contains",
		"imports",
		"unknown",
		"",
	}

	for _, typ := range validTypes {
		t.Run("valid_"+typ, func(t *testing.T) {
			if !IsValidSemanticEdgeType(typ) {
				t.Errorf("IsValidSemanticEdgeType(%q) = false, want true", typ)
			}
		})
	}

	for _, typ := range invalidTypes {
		t.Run("invalid_"+typ, func(t *testing.T) {
			if IsValidSemanticEdgeType(typ) {
				t.Errorf("IsValidSemanticEdgeType(%q) = true, want false", typ)
			}
		})
	}
}

func TestBuildSubagentPrompt(t *testing.T) {
	files := []string{"/path/to/foo.go", "/path/to/bar.go"}
	prompt := BuildSubagentPrompt(files, 1, 2, "/base")

	// Check that prompt contains expected content
	if !contains(prompt, "chunk 1 of 2") {
		t.Error("prompt should contain chunk info")
	}

	if !contains(prompt, "/path/to/foo.go") {
		t.Error("prompt should contain file paths")
	}

	if !contains(prompt, "inferred_depends") {
		t.Error("prompt should contain edge type instructions")
	}

	if !contains(prompt, "INFERRED") {
		t.Error("prompt should contain confidence level instructions")
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
