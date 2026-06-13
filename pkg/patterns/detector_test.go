package patterns

import (
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestNewDetector(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_a", Type: "function", Label: "funcA"},
	}
	edges := []*graph.Edge{
		{From: "func_a", To: "func_b", Type: "calls"},
	}

	detector := NewDetector(nodes, edges)
	if detector == nil {
		t.Fatal("Expected non-nil detector")
	}

	if len(detector.nodeMap) != 1 {
		t.Errorf("Expected 1 node in map, got %d", len(detector.nodeMap))
	}
}

func TestDetector_Detect(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_NewClient", Type: "function", Label: "NewClient", Attrs: map[string]string{
			"signature": "func NewClient(opts Options) *Client",
			"package":   "client",
		}},
		{ID: "func_handleRequest", Type: "function", Label: "handleRequest", Attrs: map[string]string{
			"signature": "func handleRequest(w http.ResponseWriter, r *http.Request)",
			"package":   "api",
		}},
		{ID: "struct_UserRepo", Type: "struct", Label: "UserRepository", Attrs: map[string]string{
			"package": "repo",
		}},
	}

	edges := []*graph.Edge{
		{From: "func_NewClient", To: "struct_Client", Type: "returns"},
	}

	detector := NewDetector(nodes, edges)
	report := detector.Detect()

	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	// Should detect factory pattern
	foundFactory := false
	for _, p := range report.Architectural {
		if p.Type == "factory" {
			foundFactory = true
			break
		}
	}
	if !foundFactory {
		t.Error("Expected to detect factory pattern for NewClient")
	}

	// Should detect handler pattern
	foundHandler := false
	for _, p := range report.Architectural {
		if p.Type == "handler" {
			foundHandler = true
			break
		}
	}
	if !foundHandler {
		t.Error("Expected to detect handler pattern for handleRequest")
	}

	// Should detect repository pattern
	foundRepo := false
	for _, p := range report.Architectural {
		if p.Type == "repository" {
			foundRepo = true
			break
		}
	}
	if !foundRepo {
		t.Error("Expected to detect repository pattern for UserRepository")
	}
}

func TestDetector_DetectGodObjects(t *testing.T) {
	// Create a struct with many methods
	nodes := []*graph.Node{
		{ID: "struct_God", Type: "struct", Label: "GodClass"},
	}

	// Add 25 method edges
	edges := []*graph.Edge{}
	for i := 0; i < 25; i++ {
		methodID := "method_" + string(rune('a'+i))
		nodes = append(nodes, &graph.Node{ID: methodID, Type: "method", Label: "method" + string(rune('A'+i))})
		edges = append(edges, &graph.Edge{From: "struct_God", To: methodID, Type: "has_method"})
	}

	detector := NewDetector(nodes, edges)
	report := detector.Detect()

	foundGod := false
	for _, ap := range report.AntiPatterns {
		if ap.Type == "god_object" {
			foundGod = true
			break
		}
	}

	if !foundGod {
		t.Error("Expected to detect god object anti-pattern")
	}
}

func TestDetector_DetectCircularDeps(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "pkg_a", Type: "package", Label: "pkgA"},
		{ID: "pkg_b", Type: "package", Label: "pkgB"},
	}

	// Circular dependency: A imports B, B imports A
	edges := []*graph.Edge{
		{From: "pkg_a", To: "pkg_b", Type: "imports"},
		{From: "pkg_b", To: "pkg_a", Type: "imports"},
	}

	detector := NewDetector(nodes, edges)
	report := detector.Detect()

	foundCircular := false
	for _, ap := range report.AntiPatterns {
		if ap.Type == "circular_dependency" {
			foundCircular = true
			break
		}
	}

	if !foundCircular {
		t.Error("Expected to detect circular dependency")
	}
}

func TestDetector_DetectHubNodes(t *testing.T) {
	// Create a hub node with many connections
	nodes := []*graph.Node{
		{ID: "hub_node", Type: "function", Label: "hubFunction"},
	}

	edges := []*graph.Edge{}
	for i := 0; i < 15; i++ {
		targetID := "target_" + string(rune('a'+i))
		nodes = append(nodes, &graph.Node{ID: targetID, Type: "function", Label: "target" + string(rune('A'+i))})
		edges = append(edges, &graph.Edge{From: "hub_node", To: targetID, Type: "calls"})
	}

	detector := NewDetector(nodes, edges)
	report := detector.Detect()

	foundHub := false
	for _, sp := range report.Structural {
		if sp.Type == "hub_node" {
			foundHub = true
			break
		}
	}

	if !foundHub {
		t.Error("Expected to detect hub node")
	}
}

func TestDetector_DetectDeadCode(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
		{ID: "func_used", Type: "function", Label: "usedFunc"},
		{ID: "func_unused", Type: "function", Label: "unusedFunc"}, // lowercase = unexported
	}

	// main calls usedFunc, but nothing calls unusedFunc
	edges := []*graph.Edge{
		{From: "func_main", To: "func_used", Type: "calls"},
	}

	detector := NewDetector(nodes, edges)
	report := detector.Detect()

	foundDead := false
	for _, ap := range report.AntiPatterns {
		if ap.Type == "dead_code" && ap.Nodes[0] == "func_unused" {
			foundDead = true
			break
		}
	}

	if !foundDead {
		t.Error("Expected to detect dead code for unusedFunc")
	}
}

func TestPatternSummary(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_NewService", Type: "function", Label: "NewService", Attrs: map[string]string{
			"signature": "func NewService() *Service",
		}},
	}

	detector := NewDetector(nodes, nil)
	report := detector.Detect()

	if report.Summary.TotalPatterns < 0 {
		t.Error("TotalPatterns should be non-negative")
	}

	if report.Summary.HealthScore < 0 || report.Summary.HealthScore > 100 {
		t.Errorf("HealthScore should be 0-100, got %f", report.Summary.HealthScore)
	}
}
