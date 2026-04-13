package obsidian

import (
	"strings"
	"testing"

	"github.com/plexusone/graphfs/pkg/analyze"
	"github.com/plexusone/graphfs/pkg/graph"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"path/to/file", "path_to_file"},
		{"file:name", "file_name"},
		{"file*name", "file_name"},
		{"file?name", "file_name"},
		{"file\"name", "file_name"},
		{"file<name>", "file_name_"},
		{"file|name", "file_name"},
		{"path\\to\\file", "path_to_file"},
		{"complex/path:with*many?bad<chars>", "complex_path_with_many_bad_chars_"},
	}

	for _, tt := range tests {
		got := SanitizeName(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateIndex(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
		{ID: "func_helper", Type: "function", Label: "helper"},
	}

	edges := []*graph.Edge{
		{From: "func_main", To: "func_helper", Type: "calls"},
	}

	godNodes := []analyze.HubNode{
		{ID: "func_main", Label: "main", Type: "function", InDegree: 0, OutDegree: 1},
	}

	communities := &analyze.ClusterResult{
		Communities: []analyze.Community{
			{ID: 1, Size: 2, Cohesion: 1.0, Members: []string{"func_main", "func_helper"}},
		},
		Modularity: 0.5,
	}

	content := GenerateIndex(nodes, edges, godNodes, communities)

	// Should contain header
	if !strings.Contains(content, "# Code Knowledge Graph") {
		t.Error("GenerateIndex should contain header")
	}

	// Should contain stats
	if !strings.Contains(content, "**Nodes:** 2") {
		t.Error("GenerateIndex should contain node count")
	}

	if !strings.Contains(content, "**Edges:** 1") {
		t.Error("GenerateIndex should contain edge count")
	}

	// Should contain modularity
	if !strings.Contains(content, "**Modularity:** 0.5") {
		t.Error("GenerateIndex should contain modularity")
	}

	// Should contain key nodes
	if !strings.Contains(content, "## Key Nodes") {
		t.Error("GenerateIndex should contain Key Nodes section")
	}

	// Should contain wikilinks
	if !strings.Contains(content, "[[nodes/func_main|main]]") {
		t.Error("GenerateIndex should contain wikilinks to nodes")
	}

	// Should contain communities
	if !strings.Contains(content, "[[communities/community-1|Community 1]]") {
		t.Error("GenerateIndex should contain wikilinks to communities")
	}
}

func TestGenerateCommunity(t *testing.T) {
	comm := analyze.Community{
		ID:       1,
		Size:     2,
		Cohesion: 0.85,
		Members:  []string{"func_main", "func_helper"},
	}

	nodeMap := map[string]*graph.Node{
		"func_main":   {ID: "func_main", Type: "function", Label: "main"},
		"func_helper": {ID: "func_helper", Type: "function", Label: "helper"},
	}

	degrees := map[string]int{
		"func_main":   5,
		"func_helper": 2,
	}

	content := GenerateCommunity(comm, nodeMap, degrees)

	// Should contain header
	if !strings.Contains(content, "# Community 1") {
		t.Error("GenerateCommunity should contain community header")
	}

	// Should contain size
	if !strings.Contains(content, "**Size:** 2 members") {
		t.Error("GenerateCommunity should contain size")
	}

	// Should contain cohesion
	if !strings.Contains(content, "**Cohesion:** 0.85") {
		t.Error("GenerateCommunity should contain cohesion")
	}

	// Should contain type section
	if !strings.Contains(content, "### Function") {
		t.Error("GenerateCommunity should contain Function type section")
	}

	// High degree nodes should have wikilinks
	if !strings.Contains(content, "[[nodes/func_main|main]]") {
		t.Error("GenerateCommunity should have wikilink for high-degree node")
	}

	// Low degree nodes should NOT have wikilinks
	if strings.Contains(content, "[[nodes/func_helper|helper]]") {
		t.Error("GenerateCommunity should not have wikilink for low-degree node")
	}

	// Should contain navigation
	if !strings.Contains(content, "[[../index|← Back to Index]]") {
		t.Error("GenerateCommunity should contain back navigation")
	}
}

func TestGenerateNode(t *testing.T) {
	node := &graph.Node{
		ID:    "func_main",
		Type:  "function",
		Label: "main",
		Attrs: map[string]string{
			"source_file": "main.go",
			"package":     "main",
		},
	}

	nodeMap := map[string]*graph.Node{
		"func_main":   node,
		"func_helper": {ID: "func_helper", Type: "function", Label: "helper"},
		"func_caller": {ID: "func_caller", Type: "function", Label: "caller"},
	}

	outgoing := map[string][]*graph.Edge{
		"func_main": {
			{From: "func_main", To: "func_helper", Type: "calls"},
		},
	}

	incoming := map[string][]*graph.Edge{
		"func_main": {
			{From: "func_caller", To: "func_main", Type: "calls"},
		},
	}

	content := GenerateNode(node, outgoing, incoming, nodeMap)

	// Should contain header
	if !strings.Contains(content, "# main") {
		t.Error("GenerateNode should contain node label as header")
	}

	// Should contain type
	if !strings.Contains(content, "**Type:** function") {
		t.Error("GenerateNode should contain type")
	}

	// Should contain ID
	if !strings.Contains(content, "**ID:** `func_main`") {
		t.Error("GenerateNode should contain ID")
	}

	// Should contain attributes
	if !strings.Contains(content, "## Attributes") {
		t.Error("GenerateNode should contain Attributes section")
	}

	if !strings.Contains(content, "**source_file:** main.go") {
		t.Error("GenerateNode should contain source_file attribute")
	}

	// Should contain references
	if !strings.Contains(content, "## References") {
		t.Error("GenerateNode should contain References section")
	}

	if !strings.Contains(content, "[[nodes/func_helper|helper]]") {
		t.Error("GenerateNode should contain wikilink to outgoing node")
	}

	// Should contain referenced by
	if !strings.Contains(content, "## Referenced By") {
		t.Error("GenerateNode should contain Referenced By section")
	}

	if !strings.Contains(content, "[[nodes/func_caller|caller]]") {
		t.Error("GenerateNode should contain wikilink to incoming node")
	}

	// Should contain navigation
	if !strings.Contains(content, "[[../index|← Back to Index]]") {
		t.Error("GenerateNode should contain back navigation")
	}
}

func TestGenerator(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
		{ID: "func_a", Type: "function", Label: "a"},
		{ID: "func_b", Type: "function", Label: "b"},
		{ID: "func_c", Type: "function", Label: "c"},
	}

	edges := []*graph.Edge{
		{From: "func_main", To: "func_a", Type: "calls"},
		{From: "func_main", To: "func_b", Type: "calls"},
		{From: "func_main", To: "func_c", Type: "calls"},
		{From: "func_a", To: "func_b", Type: "calls"},
	}

	gen := NewGenerator()
	gen.TopN = 10
	gen.MinDegree = 2

	vault := gen.Generate(nodes, edges)

	// Should have index
	if vault.Index == "" {
		t.Error("Generate should create index")
	}

	// Should have communities
	if len(vault.Communities) == 0 {
		t.Error("Generate should create communities")
	}

	// Should have nodes (those with degree >= MinDegree)
	if len(vault.Nodes) == 0 {
		t.Error("Generate should create node pages")
	}

	// func_main has degree 3, should have a page
	if _, ok := vault.Nodes["func_main"]; !ok {
		t.Error("Generate should create page for high-degree node func_main")
	}
}

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()

	if gen.TopN != 20 {
		t.Errorf("NewGenerator() TopN = %d, want 20", gen.TopN)
	}

	if gen.MinDegree != 3 {
		t.Errorf("NewGenerator() MinDegree = %d, want 3", gen.MinDegree)
	}
}
