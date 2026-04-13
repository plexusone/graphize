package metrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkSourceFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"main.go":           "package main",
		"main_test.go":      "package main",
		"pkg/util.go":       "package pkg",
		"pkg/util_test.go":  "package pkg",
		"vendor/dep.go":     "package dep",
		".hidden/secret.go": "package hidden",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("creating directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("creating file: %v", err)
		}
	}

	// Test with default options
	opts := DefaultWalkOptions()
	result, err := WalkSourceFiles(tmpDir, opts)
	if err != nil {
		t.Fatalf("WalkSourceFiles: %v", err)
	}

	// Should find main.go and pkg/util.go (not test files, vendor, or hidden)
	if len(result) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(result), result)
	}

	// Verify expected files are found
	foundMain := false
	foundUtil := false
	for _, f := range result {
		if filepath.Base(f) == "main.go" {
			foundMain = true
		}
		if filepath.Base(f) == "util.go" {
			foundUtil = true
		}
	}
	if !foundMain || !foundUtil {
		t.Errorf("missing expected files: main.go=%v, util.go=%v", foundMain, foundUtil)
	}
}

func TestWalkSourceFilesWithContent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "main.go")
	testContent := "package main\n\nfunc main() {}"
	if err := os.WriteFile(testFile, []byte(testContent), 0600); err != nil {
		t.Fatalf("creating file: %v", err)
	}

	opts := DefaultWalkOptions()
	var foundPath string
	var foundContent string

	err := WalkSourceFilesWithContent(tmpDir, opts, func(path string, content []byte) error {
		foundPath = path
		foundContent = string(content)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkSourceFilesWithContent: %v", err)
	}

	if foundPath != testFile {
		t.Errorf("expected path %q, got %q", testFile, foundPath)
	}
	if foundContent != testContent {
		t.Errorf("expected content %q, got %q", testContent, foundContent)
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path   string
		isTest bool
	}{
		{"main.go", false},
		{"main_test.go", true},
		{"util.go", false},
		{"util_test.go", true},
		{"component.test.ts", true},
		{"component.spec.ts", true},
		{"component.ts", false},
		{"UserTest.java", true},
		{"UserTests.java", true},
		{"User.java", false},
		{"src/__tests__/util.js", true},
		{"src/test/java/UserTest.java", true},
	}

	for _, tt := range tests {
		got := isTestFile(tt.path)
		if got != tt.isTest {
			t.Errorf("isTestFile(%q) = %v, want %v", tt.path, got, tt.isTest)
		}
	}
}

func TestShouldSkipDir(t *testing.T) {
	opts := DefaultWalkOptions()

	tests := []struct {
		name string
		skip bool
	}{
		{"src", false},
		{"pkg", false},
		{"vendor", true},
		{"node_modules", true},
		{"testdata", true},
		{".git", true},
		{".hidden", true},
	}

	for _, tt := range tests {
		err := shouldSkipDir(tt.name, opts)
		skipped := err == filepath.SkipDir
		if skipped != tt.skip {
			t.Errorf("shouldSkipDir(%q) skipped=%v, want %v", tt.name, skipped, tt.skip)
		}
	}
}
