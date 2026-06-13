package reuse

import (
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestNewTracker(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_a", Type: "function", Label: "funcA"},
	}
	edges := []*graph.Edge{
		{From: "func_a", To: "func_b", Type: "calls"},
	}

	tracker := NewTracker(nodes, edges)
	if tracker == nil {
		t.Fatal("Expected non-nil tracker")
	}
}

func TestTracker_Analyze(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_handleAuth", Type: "function", Label: "handleAuth", Attrs: map[string]string{
			"signature": "func handleAuth(ctx context.Context, req *Request) error",
			"package":   "auth",
		}},
		{ID: "func_handlePayment", Type: "function", Label: "handlePayment", Attrs: map[string]string{
			"signature": "func handlePayment(ctx context.Context, req *Request) error",
			"package":   "billing",
		}},
		{ID: "func_handleOrder", Type: "function", Label: "handleOrder", Attrs: map[string]string{
			"signature": "func handleOrder(ctx context.Context, req *Request) error",
			"package":   "orders",
		}},
		{ID: "struct_User", Type: "struct", Label: "User", Attrs: map[string]string{
			"package": "models",
		}},
	}

	edges := []*graph.Edge{
		{From: "func_handleAuth", To: "struct_User", Type: "uses"},
		{From: "func_handlePayment", To: "struct_User", Type: "uses"},
	}

	tracker := NewTracker(nodes, edges)
	report := tracker.Analyze()

	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	// Should find similar signatures
	if len(report.SimilarGroups) == 0 {
		t.Error("Expected to find similar function signatures")
	}

	// Check summary
	if report.Summary.TotalNodes != 4 {
		t.Errorf("Expected 4 total nodes, got %d", report.Summary.TotalNodes)
	}
}

func TestNormalizeSignature(t *testing.T) {
	tests := []struct {
		sig      string
		expected string
	}{
		{
			sig:      "func handleAuth(ctx context.Context, id string) error",
			expected: "(Context,string)->error",
		},
		{
			sig:      "func New(opts *Options) *Client",
			expected: "(Options)->Client",
		},
		{
			sig:      "func () error",
			expected: "()->error",
		},
	}

	for _, tt := range tests {
		result := normalizeSignature(tt.sig)
		if result != tt.expected {
			t.Errorf("normalizeSignature(%q) = %q, want %q", tt.sig, result, tt.expected)
		}
	}
}

func TestSimplifyType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"context.Context", "Context"},
		{"*Request", "Request"},
		{"[]string", "string"},
		{"error", "error"},
		{"int64", "int"},
		{"MyType", "MyType"},
	}

	for _, tt := range tests {
		result := simplifyType(tt.input)
		if result != tt.expected {
			t.Errorf("simplifyType(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsCommonName(t *testing.T) {
	if !isCommonName("New") {
		t.Error("Expected 'New' to be common")
	}
	if !isCommonName("Init") {
		t.Error("Expected 'Init' to be common")
	}
	if isCommonName("CustomFunction") {
		t.Error("Expected 'CustomFunction' to not be common")
	}
}

func TestFindDuplicateNames(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_a_parse", Type: "function", Label: "Parse", Attrs: map[string]string{"package": "pkgA"}},
		{ID: "func_b_parse", Type: "function", Label: "Parse", Attrs: map[string]string{"package": "pkgB"}},
		{ID: "func_c_parse", Type: "function", Label: "Parse", Attrs: map[string]string{"package": "pkgC"}},
	}

	tracker := NewTracker(nodes, nil)
	report := tracker.Analyze()

	// Should find Parse as duplicate across packages
	found := false
	for _, d := range report.DuplicateNames {
		if d.Name == "Parse" && len(d.Packages) >= 2 {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find 'Parse' as duplicate name across packages")
	}
}

func TestFindSharedDependencies(t *testing.T) {
	nodes := []*graph.Node{
		{ID: "func_a", Type: "function", Label: "funcA"},
		{ID: "func_b", Type: "function", Label: "funcB"},
		{ID: "dep_1", Type: "function", Label: "dep1"},
		{ID: "dep_2", Type: "function", Label: "dep2"},
		{ID: "dep_3", Type: "function", Label: "dep3"},
	}

	// Both func_a and func_b depend on the same 3 dependencies
	edges := []*graph.Edge{
		{From: "func_a", To: "dep_1", Type: "calls"},
		{From: "func_a", To: "dep_2", Type: "calls"},
		{From: "func_a", To: "dep_3", Type: "calls"},
		{From: "func_b", To: "dep_1", Type: "calls"},
		{From: "func_b", To: "dep_2", Type: "calls"},
		{From: "func_b", To: "dep_3", Type: "calls"},
	}

	tracker := NewTracker(nodes, edges)
	report := tracker.Analyze()

	if len(report.SharedDeps) == 0 {
		t.Error("Expected to find shared dependencies")
	}
}
