// Package provider defines the public interface for language extractors.
// External packages implement this interface to add language support to graphize.
//
// This follows the omnillm-core provider pattern, allowing external packages
// to register extractors without creating circular dependencies.
package provider

import (
	"github.com/plexusone/graphfs/pkg/graph"
)

// LanguageExtractor defines the interface for language-specific code extractors.
// Each implementation handles parsing and extraction for a specific programming language.
//
// External packages implement this interface and register via Register() in their init().
type LanguageExtractor interface {
	// Language returns the canonical name of the language (e.g., "go", "typescript", "java").
	Language() string

	// Extensions returns the file extensions this extractor handles (e.g., [".go"], [".ts", ".tsx"]).
	Extensions() []string

	// CanExtract returns true if this extractor can handle the given file path.
	// This is typically based on file extension but may include additional logic.
	CanExtract(path string) bool

	// ExtractFile extracts nodes and edges from a single source file.
	// The baseDir is used to compute relative paths for node IDs.
	ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error)

	// DetectFramework returns framework information if detected in the file.
	// Returns nil if no framework is detected.
	DetectFramework(path string) *FrameworkInfo
}

// NodeIDPrefix returns the standard prefix for node IDs for a given language.
// This ensures unique node IDs in polyglot repositories.
func NodeIDPrefix(language string) string {
	return language + "_"
}
