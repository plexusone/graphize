// Package extract provides Go AST extraction for building knowledge graphs.
// This file provides backward compatibility with the legacy Extractor API.
// New code should use MultiExtractor with the Registry pattern.
package extract

import (
	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/pkg/cache"
)

// Extractor provides backward compatibility with the original Go-only extractor API.
// It wraps MultiExtractor using the DefaultRegistry which has Go registered.
//
// Deprecated: Use MultiExtractor with Registry for multi-language support.
type Extractor struct {
	multi *MultiExtractor
}

// NewExtractor creates a new extractor using the default registry.
// This maintains backward compatibility while using the new architecture.
//
// Deprecated: Use NewMultiExtractor(DefaultRegistry) instead.
func NewExtractor() *Extractor {
	return &Extractor{
		multi: NewMultiExtractor(DefaultRegistry),
	}
}

// WithCache sets the cache for the extractor.
func (e *Extractor) WithCache(c *cache.Cache) *Extractor {
	e.multi.WithCache(c)
	return e
}

// ExtractDir extracts nodes and edges from all supported files in a directory tree.
func (e *Extractor) ExtractDir(dir string) (*graph.Graph, error) {
	return e.multi.ExtractDir(dir)
}

// ExtractDirWithStats extracts nodes and edges with cache statistics.
// Returns the legacy ExtractStats format for compatibility.
func (e *Extractor) ExtractDirWithStats(dir string) (*graph.Graph, *ExtractStats) {
	return e.multi.ExtractDirWithStats(dir)
}
