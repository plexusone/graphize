package extract

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/pkg/cache"
	"github.com/plexusone/graphize/provider"
)

// MultiExtractor coordinates extraction across multiple language extractors.
// It walks directories, dispatches files to the appropriate extractor,
// and aggregates results into a unified graph.
type MultiExtractor struct {
	registry *Registry
	cache    *cache.Cache
	// customExtractors maps extensions to custom extractors for direct injection
	customExtractors map[string]LanguageExtractor
}

// MultiExtractorOption configures a MultiExtractor.
type MultiExtractorOption func(*MultiExtractor)

// WithCustomExtractor adds a custom extractor for an extension via direct injection.
// This bypasses the global registry and allows per-instance customization.
func WithCustomExtractor(extension string, extractor LanguageExtractor) MultiExtractorOption {
	return func(m *MultiExtractor) {
		if m.customExtractors == nil {
			m.customExtractors = make(map[string]LanguageExtractor)
		}
		m.customExtractors[normalizeExtension(extension)] = extractor
	}
}

// NewMultiExtractor creates a new multi-language extractor using the given registry.
// Deprecated: Use NewMultiExtractorWithOptions for more flexibility.
func NewMultiExtractor(registry *Registry) *MultiExtractor {
	return &MultiExtractor{
		registry: registry,
	}
}

// NewMultiExtractorWithOptions creates a new multi-language extractor with options.
func NewMultiExtractorWithOptions(opts ...MultiExtractorOption) *MultiExtractor {
	m := &MultiExtractor{
		registry: DefaultRegistry,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithCache sets the cache for the extractor.
func (m *MultiExtractor) WithCache(c *cache.Cache) *MultiExtractor {
	m.cache = c
	return m
}

// getExtractor returns the appropriate extractor for a file path.
// Custom extractors take precedence over the global registry.
func (m *MultiExtractor) getExtractor(path string) LanguageExtractor {
	ext := normalizeExtension(filepath.Ext(path))

	// Check custom extractors first
	if m.customExtractors != nil {
		if extractor, ok := m.customExtractors[ext]; ok {
			return extractor
		}
	}

	// Fall back to global provider registry
	return provider.GetByPath(path)
}

// ExtractDir extracts nodes and edges from all supported files in a directory tree.
func (m *MultiExtractor) ExtractDir(dir string) (*graph.Graph, error) {
	g, _ := m.ExtractDirWithStats(dir)
	return g, nil
}

// ExtractDirWithStats extracts nodes and edges with cache and language statistics.
func (m *MultiExtractor) ExtractDirWithStats(dir string) (*graph.Graph, *ExtractStats) {
	g := graph.NewGraph()
	stats := NewExtractStats()

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories, vendor, and other excluded paths
		if info.IsDir() {
			if m.shouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if we have an extractor for this file
		extractor := m.getExtractor(path)
		if extractor == nil {
			return nil
		}

		// Skip test files based on language conventions
		if m.isTestFile(path, extractor.Language()) {
			return nil
		}

		stats.TotalFiles++
		lang := extractor.Language()
		stats.ByLanguage[lang]++

		// Get relative path for cache key
		relPath, _ := filepath.Rel(dir, path)
		if relPath == "" {
			relPath = path
		}

		// Check cache first
		if m.cache != nil {
			if cached, ok := m.cache.Get(path, relPath); ok {
				stats.CacheHits++
				for _, n := range cached.Nodes {
					g.AddNode(n)
				}
				for _, edge := range cached.Edges {
					g.AddEdge(edge)
				}
				return nil
			}
			stats.CacheMisses++
		}

		// Extract from file
		nodes, edges, err := extractor.ExtractFile(path, dir)
		if err != nil {
			stats.Errors++
			return nil
		}

		// Add to graph
		for _, n := range nodes {
			g.AddNode(n)
		}
		for _, edge := range edges {
			g.AddEdge(edge)
		}

		// Save to cache
		if m.cache != nil && len(nodes) > 0 {
			_ = m.cache.Set(path, relPath, nodes, edges)
		}

		return nil
	})

	return g, stats
}

// shouldSkipDir returns true for directories that should not be traversed.
func (m *MultiExtractor) shouldSkipDir(name string) bool {
	// Skip hidden directories
	if strings.HasPrefix(name, ".") {
		return true
	}

	// Skip common vendor/dependency directories
	skipDirs := map[string]bool{
		"vendor":       true,
		"testdata":     true,
		"node_modules": true,
		"__pycache__":  true,
		".git":         true,
		".svn":         true,
		"build":        true,
		"dist":         true,
		"target":       true, // Maven/Gradle output
		"Pods":         true, // CocoaPods
	}

	return skipDirs[name]
}

// isTestFile returns true if the file is a test file for the given language.
func (m *MultiExtractor) isTestFile(path string, language string) bool {
	base := filepath.Base(path)

	switch language {
	case "go":
		return strings.HasSuffix(path, "_test.go")

	case "typescript", "javascript":
		// *.test.ts, *.spec.ts, *.test.tsx, *.spec.tsx
		for _, suffix := range []string{".test.ts", ".spec.ts", ".test.tsx", ".spec.tsx", ".test.js", ".spec.js"} {
			if strings.HasSuffix(base, suffix) {
				return true
			}
		}
		// __tests__ directory
		if strings.Contains(path, "__tests__") {
			return true
		}

	case "java":
		// *Test.java, *Tests.java
		for _, suffix := range []string{"Test.java", "Tests.java"} {
			if strings.HasSuffix(base, suffix) {
				return true
			}
		}
		// src/test directory
		if strings.Contains(path, "/test/") || strings.Contains(path, "\\test\\") {
			return true
		}

	case "swift":
		// *Tests.swift, *Test.swift
		for _, suffix := range []string{"Tests.swift", "Test.swift"} {
			if strings.HasSuffix(base, suffix) {
				return true
			}
		}
	}

	return false
}

// DetectFrameworks scans the directory and returns detected frameworks.
func (m *MultiExtractor) DetectFrameworks(dir string) map[string][]*FrameworkInfo {
	frameworks := make(map[string][]*FrameworkInfo)

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		extractor := m.getExtractor(path)
		if extractor == nil {
			return nil
		}

		if fw := extractor.DetectFramework(path); fw != nil {
			lang := extractor.Language()
			frameworks[lang] = append(frameworks[lang], fw)
		}

		return nil
	})

	return frameworks
}

// normalizeExtension ensures the extension has a leading dot and is lowercase.
func normalizeExtension(ext string) string {
	ext = strings.ToLower(ext)
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}
