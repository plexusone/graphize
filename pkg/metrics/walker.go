package metrics

import (
	"os"
	"path/filepath"
	"strings"
)

// WalkOptions configures source file walking behavior.
type WalkOptions struct {
	// Extensions to include (e.g., []string{".go", ".java"}). Empty means all files.
	Extensions []string

	// SkipDirs are directory names to skip (e.g., "vendor", "node_modules").
	SkipDirs []string

	// SkipHidden skips directories starting with "." (default: true).
	SkipHidden bool

	// SkipTests skips test files based on language conventions.
	SkipTests bool
}

// DefaultWalkOptions returns sensible defaults for Go projects.
func DefaultWalkOptions() WalkOptions {
	return WalkOptions{
		Extensions: []string{".go"},
		SkipDirs:   []string{"vendor", "testdata", "node_modules"},
		SkipHidden: true,
		SkipTests:  true,
	}
}

// WalkSourceFiles walks a directory and returns all source files matching the options.
func WalkSourceFiles(root string, opts WalkOptions) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		if info.IsDir() {
			return shouldSkipDir(info.Name(), opts)
		}

		if shouldIncludeFile(path, opts) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// WalkSourceFilesWithContent walks a directory and calls fn for each matching file with its content.
// If fn returns an error, walking stops and the error is returned.
func WalkSourceFilesWithContent(root string, opts WalkOptions, fn func(path string, content []byte) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		if info.IsDir() {
			return shouldSkipDir(info.Name(), opts)
		}

		if !shouldIncludeFile(path, opts) {
			return nil
		}

		content, err := os.ReadFile(path) //nolint:gosec // G122: path is from filepath.Walk
		if err != nil {
			return nil // Skip unreadable files
		}

		return fn(path, content)
	})
}

// shouldSkipDir returns filepath.SkipDir if the directory should be skipped, nil otherwise.
func shouldSkipDir(name string, opts WalkOptions) error {
	// Skip hidden directories
	if opts.SkipHidden && strings.HasPrefix(name, ".") {
		return filepath.SkipDir
	}

	// Skip configured directories
	for _, skip := range opts.SkipDirs {
		if name == skip {
			return filepath.SkipDir
		}
	}

	return nil
}

// shouldIncludeFile returns true if the file should be included.
func shouldIncludeFile(path string, opts WalkOptions) bool {
	// Check extension
	if len(opts.Extensions) > 0 {
		ext := strings.ToLower(filepath.Ext(path))
		found := false
		for _, e := range opts.Extensions {
			if ext == strings.ToLower(e) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Skip test files
	if opts.SkipTests && isTestFile(path) {
		return false
	}

	return true
}

// isTestFile returns true if the file is a test file based on common conventions.
func isTestFile(path string) bool {
	base := filepath.Base(path)

	// Go: *_test.go
	if strings.HasSuffix(base, "_test.go") {
		return true
	}

	// TypeScript/JavaScript: *.test.ts, *.spec.ts, etc.
	for _, suffix := range []string{".test.ts", ".spec.ts", ".test.tsx", ".spec.tsx", ".test.js", ".spec.js"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}

	// Java: *Test.java, *Tests.java
	for _, suffix := range []string{"Test.java", "Tests.java"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}

	// In __tests__ directory
	if strings.Contains(path, "__tests__") {
		return true
	}

	// In test/ directory (Java convention)
	if strings.Contains(path, "/test/") || strings.Contains(path, "\\test\\") {
		return true
	}

	return false
}
