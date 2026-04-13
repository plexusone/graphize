// Package cache provides per-file caching for graph extraction results.
// Cache keys are based on SHA256 hashes of file content, so cached results
// are automatically invalidated when files change.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/graph"
)

// Cache manages per-file extraction caches.
type Cache struct {
	// Dir is the cache directory (typically .graphize/cache/).
	Dir string
}

// CachedExtraction stores the extraction results for a single file.
type CachedExtraction struct {
	// FileHash is the SHA256 hash of the file content.
	FileHash string `json:"file_hash"`

	// FilePath is the relative path of the source file.
	FilePath string `json:"file_path"`

	// Nodes extracted from this file.
	Nodes []*graph.Node `json:"nodes"`

	// Edges extracted from this file.
	Edges []*graph.Edge `json:"edges"`
}

// CacheStats tracks cache hit/miss statistics.
type CacheStats struct {
	Hits   int
	Misses int
}

// New creates a new cache instance.
// graphDir should be the root graph directory (e.g., ".graphize").
func New(graphDir string) *Cache {
	return &Cache{
		Dir: filepath.Join(graphDir, "cache"),
	}
}

// Init ensures the cache directory exists.
func (c *Cache) Init() error {
	return os.MkdirAll(c.Dir, 0755)
}

// Hash computes the SHA256 hash of a file's content.
func (c *Cache) Hash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// cacheKey generates a cache filename from a file path.
// Uses the relative path to create a unique, filesystem-safe key.
func (c *Cache) cacheKey(relPath string) string {
	// Hash the path to create a unique, filesystem-safe filename
	h := sha256.Sum256([]byte(relPath))
	return hex.EncodeToString(h[:16]) + ".json" // Use first 16 bytes (32 hex chars)
}

// Get retrieves a cached extraction for a file path.
// Returns nil, false if not cached or if the file has changed.
func (c *Cache) Get(path, relPath string) (*CachedExtraction, bool) {
	// Compute current file hash
	currentHash, err := c.Hash(path)
	if err != nil {
		return nil, false
	}

	// Load cached entry
	cacheFile := filepath.Join(c.Dir, c.cacheKey(relPath))
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}

	var cached CachedExtraction
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}

	// Check if file has changed
	if cached.FileHash != currentHash {
		return nil, false
	}

	return &cached, true
}

// Set stores an extraction result in the cache.
func (c *Cache) Set(path, relPath string, nodes []*graph.Node, edges []*graph.Edge) error {
	// Ensure cache directory exists
	if err := c.Init(); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	// Compute file hash
	hash, err := c.Hash(path)
	if err != nil {
		return fmt.Errorf("hashing file: %w", err)
	}

	// Create cache entry
	cached := CachedExtraction{
		FileHash: hash,
		FilePath: relPath,
		Nodes:    nodes,
		Edges:    edges,
	}

	// Serialize to JSON
	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("marshaling cache entry: %w", err)
	}

	// Write cache file
	cacheFile := filepath.Join(c.Dir, c.cacheKey(relPath))
	if err := os.WriteFile(cacheFile, data, 0600); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	return nil
}

// CheckResult contains the result of checking a file against the cache.
type CheckResult struct {
	Path    string
	RelPath string
	Cached  bool
	Entry   *CachedExtraction
}

// CheckMultiple checks multiple files against the cache.
// Returns separate lists of cached and uncached files.
func (c *Cache) CheckMultiple(baseDir string, paths []string) (cached, uncached []CheckResult) {
	for _, path := range paths {
		relPath, _ := filepath.Rel(baseDir, path)
		if relPath == "" {
			relPath = path
		}

		entry, ok := c.Get(path, relPath)
		result := CheckResult{
			Path:    path,
			RelPath: relPath,
			Cached:  ok,
			Entry:   entry,
		}

		if ok {
			cached = append(cached, result)
		} else {
			uncached = append(uncached, result)
		}
	}
	return cached, uncached
}

// Clear removes all cached entries.
func (c *Cache) Clear() error {
	return os.RemoveAll(c.Dir)
}

// Size returns the number of cached entries.
func (c *Cache) Size() (int, error) {
	entries, err := os.ReadDir(c.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			count++
		}
	}
	return count, nil
}
