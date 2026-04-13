package cypher

import (
	"strings"
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestEscapeString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"it's", "it\\'s"},
		{"line1\nline2", "line1\\nline2"},
		{"path\\to\\file", "path\\\\to\\\\file"},
		{"mixed\r\n", "mixed\\r\\n"},
		{"quote's and\\slash", "quote\\'s and\\\\slash"},
	}

	for _, tt := range tests {
		got := EscapeString(tt.input)
		if got != tt.want {
			t.Errorf("EscapeString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"valid_key", "valid_key"},
		{"key123", "key123"},
		{"key-with-dashes", "key_with_dashes"},
		{"key.with.dots", "key_with_dots"},
		{"key with spaces", "key_with_spaces"},
	}

	for _, tt := range tests {
		got := EscapeKey(tt.input)
		if got != tt.want {
			t.Errorf("EscapeKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToNeoLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"function", "Function"},
		{"method_call", "MethodCall"},
		{"struct", "Struct"},
		{"interface", "Interface"},
		{"123numeric", "N123numeric"},
		{"", "Node"},
		{"multi_word_type", "MultiWordType"},
	}

	for _, tt := range tests {
		got := ToNeoLabel(tt.input)
		if got != tt.want {
			t.Errorf("ToNeoLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToNeoRelType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"calls", "CALLS"},
		{"imports", "IMPORTS"},
		{"inferred_depends", "INFERRED_DEPENDS"},
	}

	for _, tt := range tests {
		got := ToNeoRelType(tt.input)
		if got != tt.want {
			t.Errorf("ToNeoRelType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNodeToCreate(t *testing.T) {
	node := &graph.Node{
		ID:    "func_main",
		Type:  "function",
		Label: "main",
		Attrs: map[string]string{
			"source_file": "main.go",
		},
	}

	cypher := NodeToCreate(node)

	// Should contain CREATE statement
	if !strings.HasPrefix(cypher, "CREATE (:Function {") {
		t.Errorf("NodeToCreate should start with CREATE (:Function {, got %q", cypher)
	}

	// Should contain id
	if !strings.Contains(cypher, "id: 'func_main'") {
		t.Error("NodeToCreate should contain id property")
	}

	// Should contain type
	if !strings.Contains(cypher, "type: 'function'") {
		t.Error("NodeToCreate should contain type property")
	}

	// Should contain label
	if !strings.Contains(cypher, "label: 'main'") {
		t.Error("NodeToCreate should contain label property")
	}

	// Should contain attribute
	if !strings.Contains(cypher, "source_file: 'main.go'") {
		t.Error("NodeToCreate should contain source_file attribute")
	}
}

func TestEdgeToCreate(t *testing.T) {
	edge := &graph.Edge{
		From:            "func_main",
		To:              "func_helper",
		Type:            "calls",
		Confidence:      graph.ConfidenceExtracted,
		ConfidenceScore: 1.0,
	}

	cypher := EdgeToCreate(edge)

	// Should contain MATCH and CREATE
	if !strings.HasPrefix(cypher, "MATCH (a:Node {id: 'func_main'}), (b:Node {id: 'func_helper'})") {
		t.Errorf("EdgeToCreate should start with MATCH, got %q", cypher)
	}

	// Should contain relationship type
	if !strings.Contains(cypher, "CREATE (a)-[:CALLS") {
		t.Error("EdgeToCreate should contain CALLS relationship")
	}

	// Should contain confidence
	if !strings.Contains(cypher, "confidence: 'EXTRACTED'") {
		t.Error("EdgeToCreate should contain confidence property")
	}
}

func TestGenerator(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
		{ID: "func_helper", Type: "function", Label: "helper"},
	}

	edges := []*graph.Edge{
		{From: "func_main", To: "func_helper", Type: "calls"},
	}

	gen := NewGenerator()
	cypher := gen.Generate(nodes, edges)

	// Should contain header
	if !strings.Contains(cypher, "// Neo4j Cypher import script") {
		t.Error("Generate should include header")
	}

	// Should contain constraint
	if !strings.Contains(cypher, "CREATE CONSTRAINT") {
		t.Error("Generate should include constraint")
	}

	// Should contain both nodes
	if !strings.Contains(cypher, "func_main") || !strings.Contains(cypher, "func_helper") {
		t.Error("Generate should include all nodes")
	}

	// Should contain edge
	if !strings.Contains(cypher, ":CALLS") {
		t.Error("Generate should include edge")
	}
}

func TestGeneratorNoHeader(t *testing.T) {
	gen := &Generator{
		IncludeHeader:     false,
		IncludeConstraint: false,
	}

	cypher := gen.Generate([]*graph.Node{{ID: "n1", Type: "node"}}, nil)

	if strings.Contains(cypher, "// Neo4j Cypher import script") {
		t.Error("Generate with IncludeHeader=false should not include header")
	}

	if strings.Contains(cypher, "CREATE CONSTRAINT") {
		t.Error("Generate with IncludeConstraint=false should not include constraint")
	}
}
