package htmlsite

import (
	"strings"
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/pkg/source"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()
	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.Title != "Code Graph" {
		t.Errorf("expected default title 'Code Graph', got %q", gen.Title)
	}
	if gen.DarkMode {
		t.Error("expected DarkMode to default to false")
	}
	if !gen.IncludeCommunities {
		t.Error("expected IncludeCommunities to default to true")
	}
}

func TestLoadTemplates(t *testing.T) {
	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates failed: %v", err)
	}
	if templates.Index == nil {
		t.Error("Index template is nil")
	}
	if templates.Service == nil {
		t.Error("Service template is nil")
	}
	if templates.CSS == "" {
		t.Error("CSS is empty")
	}
}

func TestGenerateSingleRepo(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
		{ID: "func_helper", Type: "function", Label: "helper"},
	}
	edges := []*graph.Edge{
		{From: "func_main", To: "func_helper", Type: "calls"},
	}

	gen := NewGenerator()
	gen.Title = "Test Graph"

	content, err := gen.Generate(nodes, edges, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check index page
	if content.Index == nil {
		t.Fatal("Index page is nil")
	}
	indexHTML := string(content.Index)
	if !strings.Contains(indexHTML, "Test Graph") {
		t.Error("Index page should contain title")
	}
	if !strings.Contains(indexHTML, "func_main") {
		t.Error("Index page should contain node ID")
	}

	// Single repo mode should not have service pages
	if len(content.Services) > 0 {
		t.Errorf("Single repo mode should not have service pages, got %d", len(content.Services))
	}
}

func TestGenerateMultiService(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "system:myapp", Type: "system", Label: "My App"},
		{ID: "svc:api", Type: "service", Label: "API Service"},
		{ID: "svc:web", Type: "service", Label: "Web Service"},
		{ID: "repo:github.com/example/api", Type: "repository", Label: "api"},
		{ID: "repo:github.com/example/web", Type: "repository", Label: "web"},
		// Code nodes
		{ID: "func_handler", Type: "function", Label: "handler", Attrs: map[string]string{"source_file": "/home/user/api/main.go"}},
		{ID: "func_render", Type: "function", Label: "render", Attrs: map[string]string{"source_file": "/home/user/web/main.go"}},
	}
	edges := []*graph.Edge{
		{From: "system:myapp", To: "svc:api", Type: "contains"},
		{From: "system:myapp", To: "svc:web", Type: "contains"},
		{From: "svc:api", To: "repo:github.com/example/api", Type: "links_to"},
		{From: "svc:web", To: "repo:github.com/example/web", Type: "links_to"},
	}

	manifest := &source.Manifest{
		Sources: []*source.Source{
			{Path: "/home/user/api"},
			{Path: "/home/user/web"},
		},
	}

	gen := NewGenerator()
	gen.Title = "Multi-Service Test"

	content, err := gen.Generate(nodes, edges, manifest)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check index page
	if content.Index == nil {
		t.Fatal("Index page is nil")
	}
	indexHTML := string(content.Index)
	if !strings.Contains(indexHTML, "Multi-Service Test") {
		t.Error("Index page should contain title")
	}
	if !strings.Contains(indexHTML, "Services") {
		t.Error("Index page should contain Services section")
	}

	// Should have service pages
	if len(content.Services) == 0 {
		t.Error("Multi-service mode should have service pages")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"API Service", "api-service"},
		{"web_app", "web-app"},
		{"My App 123", "my-app-123"},
		{"Test!@#$%", "test"},
		{"", ""},
	}

	for _, tt := range tests {
		result := slugify(tt.input)
		if result != tt.expected {
			t.Errorf("slugify(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestHasSystemNode(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []*graph.Node
		expected bool
	}{
		{
			name:     "no system node",
			nodes:    []*graph.Node{{ID: "func_main", Type: "function"}},
			expected: false,
		},
		{
			name:     "has system type",
			nodes:    []*graph.Node{{ID: "sys", Type: "system"}},
			expected: true,
		},
		{
			name:     "has system: prefix",
			nodes:    []*graph.Node{{ID: "system:myapp", Type: "other"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasSystemNode(tt.nodes)
			if result != tt.expected {
				t.Errorf("HasSystemNode() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterGraphByPath(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "n1", Type: "function", Attrs: map[string]string{"source_file": "/home/user/api/main.go"}},
		{ID: "n2", Type: "function", Attrs: map[string]string{"source_file": "/home/user/api/handler.go"}},
		{ID: "n3", Type: "function", Attrs: map[string]string{"source_file": "/home/user/web/main.go"}},
	}
	edges := []*graph.Edge{
		{From: "n1", To: "n2", Type: "calls"},
		{From: "n1", To: "n3", Type: "calls"},
	}

	filteredNodes, filteredEdges := FilterGraphByPath(nodes, edges, "/home/user/api")

	if len(filteredNodes) != 2 {
		t.Errorf("expected 2 filtered nodes, got %d", len(filteredNodes))
	}
	if len(filteredEdges) != 1 {
		t.Errorf("expected 1 filtered edge, got %d", len(filteredEdges))
	}
}

func TestCountNodeTypes(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "n1", Type: "function"},
		{ID: "n2", Type: "function"},
		{ID: "n3", Type: "class"},
	}

	counts := CountNodeTypes(nodes)

	if counts["function"] != 2 {
		t.Errorf("expected 2 functions, got %d", counts["function"])
	}
	if counts["class"] != 1 {
		t.Errorf("expected 1 class, got %d", counts["class"])
	}
}

func TestCountEdgeTypes(t *testing.T) {
	edges := []*graph.Edge{
		{From: "a", To: "b", Type: "calls"},
		{From: "b", To: "c", Type: "calls"},
		{From: "a", To: "c", Type: "imports"},
	}

	counts := CountEdgeTypes(edges)

	if counts["calls"] != 2 {
		t.Errorf("expected 2 calls, got %d", counts["calls"])
	}
	if counts["imports"] != 1 {
		t.Errorf("expected 1 imports, got %d", counts["imports"])
	}
}

func TestBuildServiceMappings(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "svc:api", Type: "service", Label: "API Service"},
		{ID: "repo:github.com/example/api", Type: "repository"},
	}
	edges := []*graph.Edge{
		{From: "svc:api", To: "repo:github.com/example/api", Type: "links_to"},
	}
	manifest := &source.Manifest{
		Sources: []*source.Source{
			{Path: "/home/user/api"},
		},
	}

	mappings := BuildServiceMappings(nodes, edges, manifest)

	if len(mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(mappings))
	}
	if mappings[0].Name != "API Service" {
		t.Errorf("expected name 'API Service', got %q", mappings[0].Name)
	}
	if mappings[0].RepoURL != "github.com/example/api" {
		t.Errorf("expected repo URL 'github.com/example/api', got %q", mappings[0].RepoURL)
	}
}

func TestGetSystemNodes(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "system:app", Type: "system"},
		{ID: "svc:api", Type: "service"},
		{ID: "repo:example", Type: "repository"},
		{ID: "func_main", Type: "function"},
	}
	edges := []*graph.Edge{
		{From: "system:app", To: "svc:api", Type: "contains"},
		{From: "svc:api", To: "func_main", Type: "contains"},
	}

	sysNodes, sysEdges := GetSystemNodes(nodes, edges)

	if len(sysNodes) != 3 {
		t.Errorf("expected 3 system nodes, got %d", len(sysNodes))
	}
	if len(sysEdges) != 1 {
		t.Errorf("expected 1 system edge, got %d", len(sysEdges))
	}
}

func TestNodesToCytoscapeJSON(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "n1", Type: "function", Label: "main"},
		{ID: "n2", Type: "function", Label: "helper"},
	}
	edges := []*graph.Edge{
		{From: "n1", To: "n2", Type: "calls"},
	}

	jsonStr, err := nodesToCytoscapeJSON(nodes, edges)
	if err != nil {
		t.Fatalf("nodesToCytoscapeJSON failed: %v", err)
	}

	if !strings.Contains(jsonStr, `"id":"n1"`) {
		t.Error("JSON should contain node ID")
	}
	if !strings.Contains(jsonStr, `"type":"function"`) {
		t.Error("JSON should contain node type")
	}
	if !strings.Contains(jsonStr, `"source":"n1"`) {
		t.Error("JSON should contain edge source")
	}
}

func TestNodesToCytoscapeJSON_OrphanEdges(t *testing.T) {
	// Test that edges referencing non-existent nodes auto-create placeholder nodes
	nodes := []*graph.Node{
		{ID: "n1", Type: "function", Label: "main"},
	}
	edges := []*graph.Edge{
		{From: "n1", To: "svc:missing", Type: "calls"}, // svc:missing doesn't exist
	}

	jsonStr, err := nodesToCytoscapeJSON(nodes, edges)
	if err != nil {
		t.Fatalf("nodesToCytoscapeJSON failed: %v", err)
	}

	// Edge should be present (not filtered)
	if !strings.Contains(jsonStr, `"source":"n1"`) {
		t.Error("Edge should be present")
	}
	// Original node should be present
	if !strings.Contains(jsonStr, `"id":"n1"`) {
		t.Error("Original node should be present")
	}
	// Placeholder node should be auto-created with label derived from ID
	if !strings.Contains(jsonStr, `"id":"svc:missing"`) {
		t.Error("Placeholder node should be created for missing endpoint")
	}
	if !strings.Contains(jsonStr, `"label":"missing"`) {
		t.Error("Placeholder node should have label derived from ID")
	}
	if !strings.Contains(jsonStr, `"placeholder":true`) {
		t.Error("Placeholder node should be marked as placeholder")
	}
}

func TestDarkModeTemplate(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
	}
	edges := []*graph.Edge{}

	gen := NewGenerator()
	gen.DarkMode = true

	content, err := gen.Generate(nodes, edges, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	indexHTML := string(content.Index)
	if !strings.Contains(indexHTML, `data-theme="dark"`) {
		t.Error("Dark mode should set data-theme attribute")
	}
}
