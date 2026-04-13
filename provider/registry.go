package provider

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Priority levels for provider registration.
// Higher priority extractors override lower priority ones for the same extension.
const (
	PriorityDefault = 0   // Default built-in providers
	PriorityThick   = 10  // SDK-based providers (can override default)
	PriorityCustom  = 100 // User-provided custom providers
)

// ExtractorFactory creates an extractor instance.
type ExtractorFactory func() LanguageExtractor

type registeredExtractor struct {
	factory  ExtractorFactory
	priority int
}

var (
	// registry maps file extensions to registered extractors
	registry   = make(map[string]registeredExtractor)
	registryMu sync.RWMutex

	// languageRegistry maps language names to registered extractors
	languageRegistry = make(map[string]registeredExtractor)
)

// Register adds an extractor factory with priority.
// Higher priority extractors override lower priority ones for the same extension.
// Call this from init() in your extractor package.
func Register(factory ExtractorFactory, priority int) {
	registryMu.Lock()
	defer registryMu.Unlock()

	ext := factory()
	lang := ext.Language()

	// Register by language name
	existing, ok := languageRegistry[lang]
	if !ok || priority >= existing.priority {
		languageRegistry[lang] = registeredExtractor{factory, priority}
	}

	// Register by each supported extension
	for _, extension := range ext.Extensions() {
		extension = normalizeExtension(extension)
		existing, ok := registry[extension]
		if !ok || priority >= existing.priority {
			registry[extension] = registeredExtractor{factory, priority}
		}
	}
}

// Get returns the extractor for a file extension.
// Returns nil if no extractor is registered for the extension.
func Get(extension string) LanguageExtractor {
	registryMu.RLock()
	defer registryMu.RUnlock()

	extension = normalizeExtension(extension)
	if reg, ok := registry[extension]; ok {
		return reg.factory()
	}
	return nil
}

// GetByPath returns the extractor for a file path.
// Returns nil if no extractor is registered for the file's extension.
func GetByPath(path string) LanguageExtractor {
	return Get(filepath.Ext(path))
}

// GetByLanguage returns the extractor for a language name.
// Returns nil if no extractor is registered for the language.
func GetByLanguage(language string) LanguageExtractor {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if reg, ok := languageRegistry[language]; ok {
		return reg.factory()
	}
	return nil
}

// Languages returns all registered language names.
func Languages() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	langs := make([]string, 0, len(languageRegistry))
	for lang := range languageRegistry {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// Extensions returns all registered file extensions.
func Extensions() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	exts := make([]string, 0, len(registry))
	for ext := range registry {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

// CanExtract returns true if there is an extractor registered for the given path.
func CanExtract(path string) bool {
	return GetByPath(path) != nil
}

// normalizeExtension ensures the extension has a leading dot and is lowercase.
func normalizeExtension(ext string) string {
	ext = strings.ToLower(ext)
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}
