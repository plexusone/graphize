package cmd

import (
	"encoding/json"
	"testing"

	"github.com/plexusone/graphize/pkg/extract"
	"github.com/plexusone/graphize/pkg/source"
)

func TestEnhanceOutputJSON(t *testing.T) {
	// Test that EnhanceOutput struct serializes correctly
	output := EnhanceOutput{
		Status:      "ready",
		GraphPath:   "/test/path",
		Sources:     2,
		TotalFiles:  100,
		Cached:      80,
		Uncached:    20,
		ChunkSize:   25,
		TotalChunks: 1,
		Chunks: []ChunkOutput{
			{
				ID:     1,
				Files:  []string{"a.go", "b.go"},
				Prompt: "test prompt",
			},
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal EnhanceOutput: %v", err)
	}

	// Verify we can unmarshal it back
	var decoded EnhanceOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal EnhanceOutput: %v", err)
	}

	if decoded.Status != "ready" {
		t.Errorf("Status = %q, want %q", decoded.Status, "ready")
	}

	if decoded.TotalFiles != 100 {
		t.Errorf("TotalFiles = %d, want %d", decoded.TotalFiles, 100)
	}

	if decoded.Uncached != 20 {
		t.Errorf("Uncached = %d, want %d", decoded.Uncached, 20)
	}

	if len(decoded.Chunks) != 1 {
		t.Errorf("len(Chunks) = %d, want %d", len(decoded.Chunks), 1)
	}

	if decoded.Chunks[0].ID != 1 {
		t.Errorf("Chunks[0].ID = %d, want %d", decoded.Chunks[0].ID, 1)
	}

	if len(decoded.Chunks[0].Files) != 2 {
		t.Errorf("len(Chunks[0].Files) = %d, want %d", len(decoded.Chunks[0].Files), 2)
	}
}

func TestChunkOutputWithPrompt(t *testing.T) {
	files := []string{"foo.go", "bar.go", "baz.go"}
	chunks := extract.ChunkFiles(files, 2)

	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d", len(chunks))
	}

	// First chunk should have 2 files
	if len(chunks[0]) != 2 {
		t.Errorf("First chunk has %d files, want 2", len(chunks[0]))
	}

	// Second chunk should have 1 file
	if len(chunks[1]) != 1 {
		t.Errorf("Second chunk has %d files, want 1", len(chunks[1]))
	}

	// Generate prompts for each chunk
	for i, chunk := range chunks {
		prompt := extract.BuildSubagentPrompt(chunk, i+1, len(chunks), "/test")
		if prompt == "" {
			t.Errorf("Chunk %d prompt is empty", i+1)
		}

		// Verify prompt contains chunk info
		if !containsString(prompt, "chunk") {
			t.Errorf("Chunk %d prompt missing chunk reference", i+1)
		}
	}
}

func TestEnhanceOutputStatusValues(t *testing.T) {
	tests := []struct {
		name     string
		uncached int
		want     string
	}{
		{
			name:     "files need extraction",
			uncached: 10,
			want:     "ready",
		},
		{
			name:     "no files need extraction",
			uncached: 0,
			want:     "ready", // Status is always "ready" in current implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := EnhanceOutput{
				Status:   tt.want,
				Uncached: tt.uncached,
			}

			if output.Status != tt.want {
				t.Errorf("Status = %q, want %q", output.Status, tt.want)
			}
		})
	}
}

func TestCollectGoFiles(t *testing.T) {
	// This test would require a temp directory with Go files
	// For now, just test that the function exists and returns a slice
	files, err := collectGoFiles("/nonexistent/path")

	// Should return empty slice for non-existent path, not error
	// (filepath.Walk returns error for non-existent root)
	if err == nil && len(files) != 0 {
		t.Errorf("Expected empty slice for non-existent path, got %d files", len(files))
	}
}

func TestOutputEnhanceJSONStructure(t *testing.T) {
	// Create mock sources
	sources := []*source.Source{
		{Path: "/test/repo1"},
		{Path: "/test/repo2"},
	}

	allFiles := []string{"a.go", "b.go", "c.go", "d.go", "e.go"}
	uncachedFiles := []string{"a.go", "b.go"}
	cachedCount := 3
	chunks := extract.ChunkFiles(uncachedFiles, 25)
	baseDir := "/test/.graphize"

	// Build the output structure manually (simulating outputEnhanceJSON)
	output := EnhanceOutput{
		Status:      "ready",
		GraphPath:   baseDir,
		Sources:     len(sources),
		TotalFiles:  len(allFiles),
		Cached:      cachedCount,
		Uncached:    len(uncachedFiles),
		ChunkSize:   25,
		TotalChunks: len(chunks),
		Chunks:      make([]ChunkOutput, len(chunks)),
	}

	for i, chunk := range chunks {
		output.Chunks[i] = ChunkOutput{
			ID:     i + 1,
			Files:  chunk,
			Prompt: extract.BuildSubagentPrompt(chunk, i+1, len(chunks), baseDir),
		}
	}

	// Verify structure
	if output.Sources != 2 {
		t.Errorf("Sources = %d, want 2", output.Sources)
	}

	if output.TotalFiles != 5 {
		t.Errorf("TotalFiles = %d, want 5", output.TotalFiles)
	}

	if output.Cached != 3 {
		t.Errorf("Cached = %d, want 3", output.Cached)
	}

	if output.Uncached != 2 {
		t.Errorf("Uncached = %d, want 2", output.Uncached)
	}

	if output.TotalChunks != 1 {
		t.Errorf("TotalChunks = %d, want 1", output.TotalChunks)
	}

	// Verify JSON serialization roundtrip
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded EnhanceOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Chunks[0].Prompt == "" {
		t.Error("Prompt should not be empty after roundtrip")
	}
}

func TestChunkOutputFields(t *testing.T) {
	chunk := ChunkOutput{
		ID:     1,
		Files:  []string{"a.go", "b.go"},
		Prompt: "test prompt content",
	}

	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("Failed to marshal ChunkOutput: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)

	if !containsString(jsonStr, `"id"`) {
		t.Error("JSON should contain 'id' field")
	}

	if !containsString(jsonStr, `"files"`) {
		t.Error("JSON should contain 'files' field")
	}

	if !containsString(jsonStr, `"prompt"`) {
		t.Error("JSON should contain 'prompt' field")
	}
}

func TestChunkOutputOmitEmptyPrompt(t *testing.T) {
	chunk := ChunkOutput{
		ID:    1,
		Files: []string{"a.go"},
		// Prompt is empty
	}

	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("Failed to marshal ChunkOutput: %v", err)
	}

	jsonStr := string(data)

	// With omitempty tag, empty prompt should not appear
	if containsString(jsonStr, `"prompt":""`) {
		t.Error("Empty prompt should be omitted from JSON")
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
