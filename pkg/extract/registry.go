// Package extract provides multi-language code extraction for building knowledge graphs.
//
// This file provides backward compatibility with the legacy Registry API.
// New code should use the provider package directly.
package extract

import (
	"github.com/plexusone/graphize/provider"
)

// Registry manages language extractors and maps file extensions to extractors.
// Deprecated: Use provider.Register, provider.Get, etc. directly.
type Registry struct{}

// NewRegistry creates a new extractor registry.
// Deprecated: The provider package uses a global registry with provider.Register.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a language extractor to the registry.
// Deprecated: Use provider.Register with a factory function instead.
func (r *Registry) Register(extractor LanguageExtractor) {
	// Wrap the extractor in a factory function
	factory := func() provider.LanguageExtractor {
		return extractor
	}
	provider.Register(factory, provider.PriorityDefault)
}

// Get returns the extractor for a given file path.
// Deprecated: Use provider.GetByPath instead.
func (r *Registry) Get(path string) LanguageExtractor {
	return provider.GetByPath(path)
}

// GetByLanguage returns the extractor for a given language name.
// Deprecated: Use provider.GetByLanguage instead.
func (r *Registry) GetByLanguage(language string) LanguageExtractor {
	return provider.GetByLanguage(language)
}

// Languages returns a list of all registered language names.
// Deprecated: Use provider.Languages instead.
func (r *Registry) Languages() []string {
	return provider.Languages()
}

// Extensions returns a list of all registered file extensions.
// Deprecated: Use provider.Extensions instead.
func (r *Registry) Extensions() []string {
	return provider.Extensions()
}

// CanExtract returns true if there is an extractor registered for the given path.
// Deprecated: Use provider.CanExtract instead.
func (r *Registry) CanExtract(path string) bool {
	return provider.CanExtract(path)
}

// DefaultRegistry is the global registry instance.
// Deprecated: Use provider.Register, provider.Get, etc. directly.
var DefaultRegistry = NewRegistry()
