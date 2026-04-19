package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/cache"
	"github.com/plexusone/graphize/pkg/extract"
	"github.com/plexusone/graphize/pkg/source"
	"github.com/spf13/cobra"

	// Import language extractors to register them with the provider registry.
	_ "github.com/plexusone/graphize/pkg/extract/golang"
	_ "github.com/plexusone/graphize/pkg/extract/java"
	_ "github.com/plexusone/graphize/pkg/extract/swift"
	_ "github.com/plexusone/graphize/pkg/extract/systemspec"
	_ "github.com/plexusone/graphize/pkg/extract/typescript"
)

var (
	analyzeNoCache  bool
	analyzeDirected bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Extract graph from tracked sources",
	Long: `Analyzes all tracked source repositories and extracts a knowledge graph.
The graph is stored in GraphFS format (one file per node/edge).

Per-file caching is used by default to skip unchanged files.
Use --no-cache to force re-extraction of all files.

Use --directed to treat edges as directed (default). This affects how
traversal and analysis interpret edges. In directed mode, 'calls' edges
flow from caller to callee. In undirected mode, edges are bidirectional.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()

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
			result := map[string]any{
				"status":  "error",
				"message": "No sources tracked. Use 'graphize add <repo>' to add sources.",
			}
			return printOutput(result)
		}

		// Create graph store
		graphStore, err := store.NewFSStore(absGraphPath)
		if err != nil {
			return fmt.Errorf("creating graph store: %w", err)
		}

		// Create extractor with optional cache
		extractor := extract.NewMultiExtractor(extract.DefaultRegistry)
		if !analyzeNoCache {
			c := cache.New(absGraphPath)
			extractor.WithCache(c)
		}

		// Track stats
		var totalNodes, totalEdges int
		var totalCacheHits, totalCacheMisses, totalFiles int
		var extractedSources []map[string]any

		// Extract from each source
		for _, src := range manifest.Sources {
			srcStart := time.Now()

			// Extract graph from source with stats
			g, stats := extractor.ExtractDirWithStats(src.Path)

			// Save to store
			if err := graphStore.SaveGraph(g); err != nil {
				return fmt.Errorf("saving graph: %w", err)
			}

			// Update source in manifest with current commit
			updated, err := source.NewSourceFromPath(src.Path)
			if err == nil {
				manifest.AddSource(updated)
			}

			srcStats := map[string]any{
				"path":         src.Path,
				"nodes":        g.NodeCount(),
				"edges":        g.EdgeCount(),
				"duration":     time.Since(srcStart).String(),
				"files":        stats.TotalFiles,
				"cache_hits":   stats.CacheHits,
				"cache_misses": stats.CacheMisses,
			}
			extractedSources = append(extractedSources, srcStats)

			totalNodes += g.NodeCount()
			totalEdges += g.EdgeCount()
			totalFiles += stats.TotalFiles
			totalCacheHits += stats.CacheHits
			totalCacheMisses += stats.CacheMisses
		}

		// Update directed flag in manifest
		manifest.Directed = analyzeDirected

		// Save updated manifest
		if err := source.SaveManifest(absGraphPath, manifest); err != nil {
			return fmt.Errorf("saving manifest: %w", err)
		}

		// Build message with cache stats
		msg := fmt.Sprintf("Extracted %d nodes and %d edges from %d source(s).", totalNodes, totalEdges, len(manifest.Sources))
		if !analyzeNoCache && totalFiles > 0 {
			hitRate := float64(totalCacheHits) / float64(totalFiles) * 100
			msg += fmt.Sprintf(" Cache: %d/%d hits (%.1f%%)", totalCacheHits, totalFiles, hitRate)
		}

		result := map[string]any{
			"status":       "success",
			"graph_path":   absGraphPath,
			"total_nodes":  totalNodes,
			"total_edges":  totalEdges,
			"total_files":  totalFiles,
			"cache_hits":   totalCacheHits,
			"cache_misses": totalCacheMisses,
			"directed":     analyzeDirected,
			"sources":      extractedSources,
			"duration":     time.Since(startTime).String(),
			"message":      msg,
		}
		return printOutput(result)
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().BoolVar(&analyzeNoCache, "no-cache", false, "Disable caching, re-extract all files")
	analyzeCmd.Flags().BoolVar(&analyzeDirected, "directed", true, "Treat graph as directed (edges flow from->to)")
}
