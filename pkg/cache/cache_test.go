package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

func TestCache_HashConsistency(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	cache := New(filepath.Join(tmpDir, ".graphize"))

	// Hash should be consistent
	hash1, err := cache.Hash(testFile)
	if err != nil {
		t.Fatalf("hashing file (1): %v", err)
	}

	hash2, err := cache.Hash(testFile)
	if err != nil {
		t.Fatalf("hashing file (2): %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("hash inconsistent: %s != %s", hash1, hash2)
	}

	// Hash should change when file content changes
	newContent := []byte("package main\n\nfunc main() { println(\"hello\") }\n")
	if err := os.WriteFile(testFile, newContent, 0600); err != nil {
		t.Fatalf("updating test file: %v", err)
	}

	hash3, err := cache.Hash(testFile)
	if err != nil {
		t.Fatalf("hashing file (3): %v", err)
	}

	if hash1 == hash3 {
		t.Errorf("hash should change after file modification")
	}
}

func TestCache_GetSet(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	cache := New(filepath.Join(tmpDir, ".graphize"))

	// Initially, cache should miss
	_, ok := cache.Get(testFile, "test.go")
	if ok {
		t.Error("expected cache miss for new file")
	}

	// Set cache entry
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
	}
	edges := []*graph.Edge{
		{From: "file_test.go", To: "func_main", Type: "contains"},
	}

	if err := cache.Set(testFile, "test.go", nodes, edges); err != nil {
		t.Fatalf("setting cache: %v", err)
	}

	// Now cache should hit
	cached, ok := cache.Get(testFile, "test.go")
	if !ok {
		t.Error("expected cache hit after set")
	}

	if len(cached.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(cached.Nodes))
	}

	if len(cached.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(cached.Edges))
	}

	if cached.Nodes[0].ID != "func_main" {
		t.Errorf("expected node ID 'func_main', got '%s'", cached.Nodes[0].ID)
	}
}

func TestCache_Invalidation(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	cache := New(filepath.Join(tmpDir, ".graphize"))

	// Set cache entry
	nodes := []*graph.Node{
		{ID: "func_main", Type: "function", Label: "main"},
	}
	if err := cache.Set(testFile, "test.go", nodes, nil); err != nil {
		t.Fatalf("setting cache: %v", err)
	}

	// Cache should hit
	_, ok := cache.Get(testFile, "test.go")
	if !ok {
		t.Error("expected cache hit after set")
	}

	// Modify the file
	newContent := []byte("package main\n\nfunc main() { println(\"hello\") }\n")
	if err := os.WriteFile(testFile, newContent, 0600); err != nil {
		t.Fatalf("updating test file: %v", err)
	}

	// Cache should miss (file changed)
	_, ok = cache.Get(testFile, "test.go")
	if ok {
		t.Error("expected cache miss after file modification")
	}
}

func TestCache_CheckMultiple(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.go")
	file2 := filepath.Join(tmpDir, "file2.go")
	file3 := filepath.Join(tmpDir, "file3.go")

	for _, f := range []string{file1, file2, file3} {
		if err := os.WriteFile(f, []byte("package main"), 0600); err != nil {
			t.Fatalf("creating test file: %v", err)
		}
	}

	cache := New(filepath.Join(tmpDir, ".graphize"))

	// Cache some files
	if err := cache.Set(file1, "file1.go", nil, nil); err != nil {
		t.Fatalf("caching file1: %v", err)
	}
	if err := cache.Set(file3, "file3.go", nil, nil); err != nil {
		t.Fatalf("caching file3: %v", err)
	}

	// Check multiple
	paths := []string{file1, file2, file3}
	cached, uncached := cache.CheckMultiple(tmpDir, paths)

	if len(cached) != 2 {
		t.Errorf("expected 2 cached, got %d", len(cached))
	}

	if len(uncached) != 1 {
		t.Errorf("expected 1 uncached, got %d", len(uncached))
	}

	// file2 should be uncached
	if len(uncached) > 0 && uncached[0].RelPath != "file2.go" {
		t.Errorf("expected file2.go to be uncached, got %s", uncached[0].RelPath)
	}
}

func TestCache_Size(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cache := New(filepath.Join(tmpDir, ".graphize"))

	// Initially empty
	size, err := cache.Size()
	if err != nil {
		t.Fatalf("getting size: %v", err)
	}
	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}

	// Add entries
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(tmpDir, "file"+string(rune('1'+i))+".go")
		if err := os.WriteFile(testFile, []byte("package main"), 0600); err != nil {
			t.Fatalf("creating test file: %v", err)
		}
		if err := cache.Set(testFile, filepath.Base(testFile), nil, nil); err != nil {
			t.Fatalf("caching file: %v", err)
		}
	}

	size, err = cache.Size()
	if err != nil {
		t.Fatalf("getting size: %v", err)
	}
	if size != 3 {
		t.Errorf("expected size 3, got %d", size)
	}
}

func TestCache_Clear(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cache := New(filepath.Join(tmpDir, ".graphize"))

	// Add an entry
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}
	if err := cache.Set(testFile, "test.go", nil, nil); err != nil {
		t.Fatalf("caching file: %v", err)
	}

	// Clear
	if err := cache.Clear(); err != nil {
		t.Fatalf("clearing cache: %v", err)
	}

	// Cache should be empty
	size, err := cache.Size()
	if err != nil {
		t.Fatalf("getting size: %v", err)
	}
	if size != 0 {
		t.Errorf("expected size 0 after clear, got %d", size)
	}
}
