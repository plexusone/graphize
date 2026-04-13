// Package extract provides multi-language code extraction for building knowledge graphs.
//
// This package provides backward compatibility by re-exporting types from the
// provider package. New code should import github.com/plexusone/graphize/provider
// directly for the public interface.
package extract

import (
	"github.com/plexusone/graphize/provider"
)

// LanguageExtractor is an alias for provider.LanguageExtractor.
// Deprecated: Use provider.LanguageExtractor directly.
type LanguageExtractor = provider.LanguageExtractor

// FrameworkInfo is an alias for provider.FrameworkInfo.
// Deprecated: Use provider.FrameworkInfo directly.
type FrameworkInfo = provider.FrameworkInfo

// ExtractStats is an alias for provider.ExtractStats.
// Deprecated: Use provider.ExtractStats directly.
type ExtractStats = provider.ExtractStats

// NewExtractStats creates a new ExtractStats instance.
// Deprecated: Use provider.NewExtractStats directly.
func NewExtractStats() *ExtractStats {
	return provider.NewExtractStats()
}

// NodeIDPrefix returns the standard prefix for node IDs for a given language.
// Deprecated: Use provider.NodeIDPrefix directly.
func NodeIDPrefix(language string) string {
	return provider.NodeIDPrefix(language)
}
