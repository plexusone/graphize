package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphize/pkg/cache"
	"github.com/plexusone/graphize/pkg/extract"
	"github.com/plexusone/graphize/pkg/metrics"
	"github.com/plexusone/graphize/pkg/source"
	"github.com/spf13/cobra"
)

var (
	enhanceForce     bool
	enhanceChunkSize int
	enhanceSource    string
	enhancePrompt    bool
	enhanceJSON      bool
)

var enhanceCmd = &cobra.Command{
	Use:   "enhance",
	Short: "Prepare files for LLM semantic extraction",
	Long: `Prepare Go source files for LLM semantic extraction.

This command identifies files that need semantic analysis and outputs them
in a format suitable for the /graphize enhance skill or multi-agent-spec subagents.

The actual LLM extraction should be performed by an AI agent using the
semantic-extractor subagent spec in agents/specs/.

Use --force to ignore cache and re-analyze all files.

Examples:
  graphize enhance              # List uncached files
  graphize enhance --force      # List all files (ignore cache)
  graphize enhance --chunk-size 30  # Use larger chunks`,
	RunE: runEnhance,
}

func init() {
	rootCmd.AddCommand(enhanceCmd)
	enhanceCmd.Flags().BoolVar(&enhanceForce, "force", false, "Ignore cache, re-extract all files")
	enhanceCmd.Flags().IntVar(&enhanceChunkSize, "chunk-size", 25, "Files per chunk for parallel processing")
	enhanceCmd.Flags().StringVar(&enhanceSource, "source", "", "Only analyze specific source path")
	enhanceCmd.Flags().BoolVar(&enhancePrompt, "prompt", false, "Output subagent prompts for each chunk")
	enhanceCmd.Flags().BoolVar(&enhanceJSON, "json", false, "Output in JSON format for automation")
}

func runEnhance(cmd *cobra.Command, args []string) error {
	// Resolve graph path
	absGraphPath, err := filepath.Abs(graphPath)
	if err != nil {
		return fmt.Errorf("resolving graph path: %w", err)
	}

	// Load manifest
	manifest, err := source.LoadManifest(absGraphPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if len(manifest.Sources) == 0 {
		return fmt.Errorf("no sources tracked. Run 'graphize add <repo>' first")
	}

	// Filter sources if specific one requested
	sources := manifest.Sources
	if enhanceSource != "" {
		var filtered []*source.Source
		for _, src := range sources {
			if src.Path == enhanceSource || strings.HasSuffix(src.Path, enhanceSource) {
				filtered = append(filtered, src)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("source not found: %s", enhanceSource)
		}
		sources = filtered
	}

	// Create cache for checking
	c := cache.New(absGraphPath)

	// Collect all Go files
	var allFiles []string
	for _, src := range sources {
		files, err := collectGoFiles(src.Path)
		if err != nil {
			return fmt.Errorf("collecting files from %s: %w", src.Path, err)
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		fmt.Println("No Go files found to analyze.")
		return nil
	}

	// Check cache for each file
	var uncachedFiles []string
	var cachedCount int

	if enhanceForce {
		uncachedFiles = allFiles
	} else {
		for _, path := range allFiles {
			relPath, _ := filepath.Rel(absGraphPath, path)
			if relPath == "" {
				relPath = path
			}
			if _, ok := c.Get(path, relPath); ok {
				cachedCount++
			} else {
				uncachedFiles = append(uncachedFiles, path)
			}
		}
	}

	// Build chunks
	chunks := extract.ChunkFiles(uncachedFiles, enhanceChunkSize)

	// JSON output mode - clean output, no headers
	if enhanceJSON {
		return outputEnhanceJSON(sources, allFiles, uncachedFiles, cachedCount, chunks, absGraphPath)
	}

	// Prompt output mode - clean output for piping
	if enhancePrompt {
		if len(uncachedFiles) == 0 {
			fmt.Println("# No files need semantic extraction (all cached)")
			return nil
		}
		return outputEnhancePrompts(chunks, absGraphPath)
	}

	// Default: human-readable output with summary
	fmt.Printf("Graphize Enhance - Semantic Extraction Prep\n")
	fmt.Printf("============================================\n\n")
	fmt.Printf("Sources: %d\n", len(sources))
	fmt.Printf("Total Go files: %d\n", len(allFiles))
	fmt.Printf("Cached (unchanged): %d\n", cachedCount)
	fmt.Printf("Need extraction: %d\n", len(uncachedFiles))
	fmt.Printf("Chunk size: %d\n", enhanceChunkSize)

	if len(uncachedFiles) == 0 {
		fmt.Println("\nAll files are cached. Use --force to re-extract.")
		return nil
	}

	numChunks := len(chunks)
	fmt.Printf("Chunks needed: %d\n\n", numChunks)

	fmt.Printf("Files to analyze (by chunk):\n")
	fmt.Printf("----------------------------\n")

	for i, chunk := range chunks {
		fmt.Printf("\nChunk %d/%d (%d files):\n", i+1, numChunks, len(chunk))
		for _, f := range chunk {
			fmt.Printf("  - %s\n", f)
		}
	}

	fmt.Printf("\n----------------------------\n")
	fmt.Printf("To run semantic extraction:\n")
	fmt.Printf("  1. Use /semantic-extract skill in Claude Code\n")
	fmt.Printf("  2. Or run: graphize enhance --prompt | for automation\n")
	fmt.Printf("  3. Or run: graphize enhance --json | for scripting\n")

	return nil
}

// EnhanceOutput represents the JSON output of the enhance command.
type EnhanceOutput struct {
	Status      string        `json:"status"`
	GraphPath   string        `json:"graph_path"`
	Sources     int           `json:"sources"`
	TotalFiles  int           `json:"total_files"`
	Cached      int           `json:"cached"`
	Uncached    int           `json:"uncached"`
	ChunkSize   int           `json:"chunk_size"`
	TotalChunks int           `json:"total_chunks"`
	Chunks      []ChunkOutput `json:"chunks"`
}

// ChunkOutput represents a single chunk in JSON output.
type ChunkOutput struct {
	ID     int      `json:"id"`
	Files  []string `json:"files"`
	Prompt string   `json:"prompt,omitempty"`
}

func outputEnhanceJSON(sources []*source.Source, allFiles, uncachedFiles []string, cachedCount int, chunks [][]string, baseDir string) error {
	output := EnhanceOutput{
		Status:      "ready",
		GraphPath:   baseDir,
		Sources:     len(sources),
		TotalFiles:  len(allFiles),
		Cached:      cachedCount,
		Uncached:    len(uncachedFiles),
		ChunkSize:   enhanceChunkSize,
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

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func outputEnhancePrompts(chunks [][]string, baseDir string) error {
	numChunks := len(chunks)

	fmt.Printf("# Graphize Semantic Extraction Prompts\n")
	fmt.Printf("# Total chunks: %d\n", numChunks)
	fmt.Printf("# Base directory: %s\n\n", baseDir)

	for i, chunk := range chunks {
		fmt.Printf("## CHUNK %d/%d\n\n", i+1, numChunks)
		fmt.Printf("Files: %d\n", len(chunk))
		for _, f := range chunk {
			fmt.Printf("  - %s\n", f)
		}
		fmt.Printf("\n### Prompt\n\n")
		fmt.Println(extract.BuildSubagentPrompt(chunk, i+1, numChunks, baseDir))
		fmt.Printf("\n---\n\n")
	}

	return nil
}

// collectGoFiles walks a directory and returns all .go files (excluding tests, vendor, etc.)
func collectGoFiles(dir string) ([]string, error) {
	opts := metrics.WalkOptions{
		Extensions: []string{".go"},
		SkipDirs:   []string{"vendor", "testdata"},
		SkipHidden: true,
		SkipTests:  true,
	}
	return metrics.WalkSourceFiles(dir, opts)
}
