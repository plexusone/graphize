package systemspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractorInterface(t *testing.T) {
	e := New()

	if e.Language() != "system-spec" {
		t.Errorf("expected language 'system-spec', got %q", e.Language())
	}

	exts := e.Extensions()
	if len(exts) != 1 || exts[0] != ".json" {
		t.Errorf("expected extensions ['.json'], got %v", exts)
	}

	// Non-JSON file should not be extractable
	if e.CanExtract("file.go") {
		t.Error("should not extract .go files")
	}

	// DetectFramework should return nil
	if e.DetectFramework("any.json") != nil {
		t.Error("expected nil framework info")
	}
}

func TestExtractFile(t *testing.T) {
	// Create a temporary system-spec file
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "system.json")

	specContent := `{
		"name": "test-system",
		"description": "Test system for unit tests",
		"services": {
			"api": {
				"image": { "name": "api", "tag": "v1.0" },
				"repo": { "url": "https://github.com/org/api" },
				"connections": {
					"db-service": { "port": 5432, "protocol": "postgres" }
				}
			},
			"db-service": {
				"image": { "name": "postgres", "tag": "15" },
				"aws": {
					"rds": [{ "name": "main-db", "engine": "postgres", "port": 5432 }]
				}
			}
		}
	}`

	if err := os.WriteFile(specPath, []byte(specContent), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	e := New()

	// Should be able to extract
	if !e.CanExtract(specPath) {
		t.Fatal("should be able to extract system-spec file")
	}

	nodes, edges, err := e.ExtractFile(specPath, tmpDir)
	if err != nil {
		t.Fatalf("ExtractFile failed: %v", err)
	}

	// Verify nodes: 1 system + 2 services + 1 RDS = 4
	if len(nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(nodes))
		for _, n := range nodes {
			t.Logf("  node: %s (%s)", n.ID, n.Type)
		}
	}

	// Verify we have the expected node types
	nodeTypes := make(map[string]int)
	for _, n := range nodes {
		nodeTypes[n.Type]++
	}

	if nodeTypes["system"] != 1 {
		t.Errorf("expected 1 system node, got %d", nodeTypes["system"])
	}
	if nodeTypes["service"] != 2 {
		t.Errorf("expected 2 service nodes, got %d", nodeTypes["service"])
	}
	if nodeTypes["database"] != 1 {
		t.Errorf("expected 1 database node, got %d", nodeTypes["database"])
	}

	// Verify edges include links_to
	hasLinksTo := false
	for _, edge := range edges {
		if edge.Type == "links_to" {
			hasLinksTo = true
			break
		}
	}
	if !hasLinksTo {
		t.Error("expected at least one links_to edge for service with repo")
	}
}

func TestCanExtractNonSystemSpec(t *testing.T) {
	// Create a regular JSON file that is NOT a system-spec
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")

	// This JSON has no "name" and "services" at root
	configContent := `{"key": "value", "nested": {"foo": "bar"}}`

	if err := os.WriteFile(jsonPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	e := New()

	// Should NOT be able to extract regular JSON
	if e.CanExtract(jsonPath) {
		t.Error("should not extract non-system-spec JSON files")
	}
}
