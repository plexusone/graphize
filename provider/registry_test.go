package provider

import (
	"testing"

	"github.com/plexusone/graphfs/pkg/graph"
)

// mockExtractor is a test implementation of LanguageExtractor.
type mockExtractor struct {
	language   string
	extensions []string
}

func (m *mockExtractor) Language() string            { return m.language }
func (m *mockExtractor) Extensions() []string        { return m.extensions }
func (m *mockExtractor) CanExtract(path string) bool { return true }
func (m *mockExtractor) ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error) {
	return nil, nil, nil
}
func (m *mockExtractor) DetectFramework(path string) *FrameworkInfo { return nil }

func TestRegisterAndGet(t *testing.T) {
	// Register a test extractor
	Register(func() LanguageExtractor {
		return &mockExtractor{
			language:   "test",
			extensions: []string{".test"},
		}
	}, PriorityDefault)

	// Get by extension
	ext := Get(".test")
	if ext == nil {
		t.Fatal("expected extractor for .test extension")
	}
	if ext.Language() != "test" {
		t.Errorf("expected language 'test', got '%s'", ext.Language())
	}

	// Get by path
	ext = GetByPath("/path/to/file.test")
	if ext == nil {
		t.Fatal("expected extractor for .test path")
	}

	// Get by language
	ext = GetByLanguage("test")
	if ext == nil {
		t.Fatal("expected extractor for 'test' language")
	}
}

func TestPriorityOverride(t *testing.T) {
	// Register a default extractor
	Register(func() LanguageExtractor {
		return &mockExtractor{
			language:   "priority_default",
			extensions: []string{".priority"},
		}
	}, PriorityDefault)

	// Register a thick extractor (higher priority)
	Register(func() LanguageExtractor {
		return &mockExtractor{
			language:   "priority_thick",
			extensions: []string{".priority"},
		}
	}, PriorityThick)

	// The thick extractor should override
	ext := Get(".priority")
	if ext == nil {
		t.Fatal("expected extractor for .priority extension")
	}
	if ext.Language() != "priority_thick" {
		t.Errorf("expected language 'priority_thick', got '%s'", ext.Language())
	}

	// Register a custom extractor (highest priority)
	Register(func() LanguageExtractor {
		return &mockExtractor{
			language:   "priority_custom",
			extensions: []string{".priority"},
		}
	}, PriorityCustom)

	ext = Get(".priority")
	if ext.Language() != "priority_custom" {
		t.Errorf("expected language 'priority_custom', got '%s'", ext.Language())
	}
}

func TestLowerPriorityDoesNotOverride(t *testing.T) {
	// Register a custom extractor (high priority)
	Register(func() LanguageExtractor {
		return &mockExtractor{
			language:   "keep_custom",
			extensions: []string{".keep"},
		}
	}, PriorityCustom)

	// Try to register a default extractor (lower priority)
	Register(func() LanguageExtractor {
		return &mockExtractor{
			language:   "keep_default",
			extensions: []string{".keep"},
		}
	}, PriorityDefault)

	// The custom extractor should remain
	ext := Get(".keep")
	if ext == nil {
		t.Fatal("expected extractor for .keep extension")
	}
	if ext.Language() != "keep_custom" {
		t.Errorf("expected language 'keep_custom', got '%s'", ext.Language())
	}
}

func TestLanguages(t *testing.T) {
	langs := Languages()
	if len(langs) == 0 {
		t.Fatal("expected at least one registered language")
	}

	// Should contain our test languages
	found := false
	for _, lang := range langs {
		if lang == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'test' in languages list")
	}
}

func TestExtensions(t *testing.T) {
	exts := Extensions()
	if len(exts) == 0 {
		t.Fatal("expected at least one registered extension")
	}

	// Should contain our test extension
	found := false
	for _, ext := range exts {
		if ext == ".test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected '.test' in extensions list")
	}
}

func TestCanExtract(t *testing.T) {
	if !CanExtract("file.test") {
		t.Error("expected CanExtract to return true for .test files")
	}

	if CanExtract("file.nonexistent") {
		t.Error("expected CanExtract to return false for unregistered extension")
	}
}

func TestNormalizeExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{".go", ".go"},
		{"go", ".go"},
		{".GO", ".go"},
		{"GO", ".go"},
		{"", ""},
	}

	for _, tt := range tests {
		result := normalizeExtension(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeExtension(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
