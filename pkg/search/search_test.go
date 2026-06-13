package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestNewSearcher(t *testing.T) {
	tmpDir := t.TempDir()
	graphDir := filepath.Join(tmpDir, ".graphize")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatal(err)
	}

	searcher, err := NewSearcher(graphDir)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}
	defer searcher.Close()

	// Check stats
	stats, err := searcher.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.TotalDocs != 0 {
		t.Errorf("Expected 0 docs, got %d", stats.TotalDocs)
	}
}

func TestSearcher_IndexAndSearch(t *testing.T) {
	tmpDir := t.TempDir()
	graphDir := filepath.Join(tmpDir, ".graphize")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatal(err)
	}

	searcher, err := NewSearcher(graphDir)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}
	defer searcher.Close()

	// Create test nodes
	nodes := []*graph.Node{
		{
			ID:    "func_handleAuth",
			Type:  "function",
			Label: "handleAuth",
			Attrs: map[string]string{
				"doc":         "handleAuth handles user authentication and validates credentials",
				"package":     "auth",
				"source_file": "pkg/auth/handler.go",
				"signature":   "func handleAuth(ctx context.Context, req *AuthRequest) error",
			},
		},
		{
			ID:    "func_validateToken",
			Type:  "function",
			Label: "validateToken",
			Attrs: map[string]string{
				"doc":         "validateToken verifies JWT tokens and extracts claims",
				"package":     "auth",
				"source_file": "pkg/auth/token.go",
			},
		},
		{
			ID:    "struct_User",
			Type:  "struct",
			Label: "User",
			Attrs: map[string]string{
				"doc":         "User represents a registered user in the system",
				"package":     "models",
				"source_file": "pkg/models/user.go",
			},
		},
		{
			ID:    "func_handlePayment",
			Type:  "function",
			Label: "handlePayment",
			Attrs: map[string]string{
				"doc":         "handlePayment processes credit card payments",
				"package":     "billing",
				"source_file": "pkg/billing/payment.go",
			},
		},
	}

	// Index nodes
	if err := searcher.IndexNodes(nodes); err != nil {
		t.Fatalf("IndexNodes failed: %v", err)
	}

	// Verify index count
	stats, _ := searcher.Stats()
	if stats.TotalDocs != 4 {
		t.Errorf("Expected 4 docs, got %d", stats.TotalDocs)
	}

	// Test basic search
	t.Run("basic search", func(t *testing.T) {
		result, err := searcher.Search("authentication", SearchOptions{Limit: 10})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if result.TotalHits == 0 {
			t.Error("Expected at least one hit for 'authentication'")
		}

		if result.Query != "authentication" {
			t.Errorf("Expected query 'authentication', got %s", result.Query)
		}
	})

	// Test type filter
	t.Run("type filter", func(t *testing.T) {
		result, err := searcher.Search("handle", SearchOptions{
			Limit:     10,
			NodeTypes: []string{"function"},
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find handleAuth and handlePayment but not User struct
		for _, r := range result.Results {
			if r.Type != "function" {
				t.Errorf("Expected only functions, got type %s", r.Type)
			}
		}
	})

	// Test fuzzy search
	t.Run("fuzzy search", func(t *testing.T) {
		// Search with typo (intentional misspelling)
		result, err := searcher.Search("authenticaton", SearchOptions{ //nolint:misspell // Intentional typo for fuzzy test
			Limit:     10,
			FuzzyDist: 2,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if result.TotalHits == 0 {
			t.Error("Expected fuzzy match for 'authenticaton' (typo)")
		}
	})

	// Test package search
	t.Run("package search", func(t *testing.T) {
		result, err := searcher.Search("auth", SearchOptions{Limit: 10})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if result.TotalHits < 2 {
			t.Errorf("Expected at least 2 hits for 'auth' package, got %d", result.TotalHits)
		}
	})

	// Test no results
	t.Run("no results", func(t *testing.T) {
		result, err := searcher.Search("xyznonexistent", SearchOptions{Limit: 10})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if result.TotalHits != 0 {
			t.Errorf("Expected 0 hits, got %d", result.TotalHits)
		}
	})
}

func TestSearcher_Reindex(t *testing.T) {
	tmpDir := t.TempDir()
	graphDir := filepath.Join(tmpDir, ".graphize")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatal(err)
	}

	searcher, err := NewSearcher(graphDir)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}
	defer searcher.Close()

	// Index initial nodes
	nodes1 := []*graph.Node{
		{ID: "func_a", Type: "function", Label: "funcA"},
		{ID: "func_b", Type: "function", Label: "funcB"},
	}
	if err := searcher.IndexNodes(nodes1); err != nil {
		t.Fatalf("IndexNodes failed: %v", err)
	}

	stats1, _ := searcher.Stats()
	if stats1.TotalDocs != 2 {
		t.Errorf("Expected 2 docs, got %d", stats1.TotalDocs)
	}

	// Reindex with different nodes
	nodes2 := []*graph.Node{
		{ID: "func_c", Type: "function", Label: "funcC"},
		{ID: "func_d", Type: "function", Label: "funcD"},
		{ID: "func_e", Type: "function", Label: "funcE"},
	}
	if err := searcher.Reindex(nodes2); err != nil {
		t.Fatalf("Reindex failed: %v", err)
	}

	stats2, _ := searcher.Stats()
	if stats2.TotalDocs != 3 {
		t.Errorf("Expected 3 docs after reindex, got %d", stats2.TotalDocs)
	}

	// Verify old nodes are gone
	result, _ := searcher.Search("funcA", SearchOptions{Limit: 10})
	if result.TotalHits != 0 {
		t.Error("Expected funcA to be removed after reindex")
	}

	// Verify new nodes are searchable
	result, _ = searcher.Search("funcC", SearchOptions{Limit: 10})
	if result.TotalHits == 0 {
		t.Error("Expected funcC to be found after reindex")
	}
}

func TestNodeToIndexed(t *testing.T) {
	node := &graph.Node{
		ID:    "func_test",
		Type:  "function",
		Label: "testFunction",
		Attrs: map[string]string{
			"doc":         "This is a test function",
			"package":     "mypackage",
			"source_file": "test.go",
			"signature":   "func testFunction() error",
		},
	}

	indexed := nodeToIndexed(node)

	if indexed.ID != "func_test" {
		t.Errorf("Expected ID 'func_test', got %s", indexed.ID)
	}
	if indexed.Type != "function" {
		t.Errorf("Expected Type 'function', got %s", indexed.Type)
	}
	if indexed.Label != "testFunction" {
		t.Errorf("Expected Label 'testFunction', got %s", indexed.Label)
	}
	if indexed.Doc != "This is a test function" {
		t.Errorf("Expected Doc 'This is a test function', got %s", indexed.Doc)
	}
	if indexed.Package != "mypackage" {
		t.Errorf("Expected Package 'mypackage', got %s", indexed.Package)
	}
	if indexed.SourceFile != "test.go" {
		t.Errorf("Expected SourceFile 'test.go', got %s", indexed.SourceFile)
	}
	if indexed.Signature != "func testFunction() error" {
		t.Errorf("Expected Signature, got %s", indexed.Signature)
	}
	if indexed.AllText == "" {
		t.Error("Expected AllText to be populated")
	}
}

func TestSearchOptions_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	graphDir := filepath.Join(tmpDir, ".graphize")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatal(err)
	}

	searcher, err := NewSearcher(graphDir)
	if err != nil {
		t.Fatalf("NewSearcher failed: %v", err)
	}
	defer searcher.Close()

	// Index a node
	nodes := []*graph.Node{
		{ID: "func_test", Type: "function", Label: "test"},
	}
	if err := searcher.IndexNodes(nodes); err != nil {
		t.Fatal(err)
	}

	// Search with empty options (should use defaults)
	result, err := searcher.Search("test", SearchOptions{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Default limit should be applied
	if result.Query != "test" {
		t.Errorf("Expected query 'test', got %s", result.Query)
	}
}
