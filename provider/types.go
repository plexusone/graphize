package provider

// FrameworkInfo describes a detected framework in source code.
type FrameworkInfo struct {
	// Name is the canonical framework name (e.g., "spring", "express", "rails").
	Name string `json:"name"`

	// Version is the detected version if available.
	Version string `json:"version,omitempty"`

	// Layer indicates the architectural layer (e.g., "controller", "service", "repository").
	Layer string `json:"layer,omitempty"`

	// Annotations are framework-specific annotations found (for Java/Kotlin).
	Annotations []string `json:"annotations,omitempty"`

	// Conventions are framework-specific conventions matched (for Rails, etc.).
	Conventions []string `json:"conventions,omitempty"`
}

// ExtractStats tracks extraction statistics across multiple files.
type ExtractStats struct {
	TotalFiles  int            `json:"total_files"`
	CacheHits   int            `json:"cache_hits"`
	CacheMisses int            `json:"cache_misses"`
	Errors      int            `json:"errors"`
	ByLanguage  map[string]int `json:"by_language,omitempty"`
}

// NewExtractStats creates a new ExtractStats instance.
func NewExtractStats() *ExtractStats {
	return &ExtractStats{
		ByLanguage: make(map[string]int),
	}
}
