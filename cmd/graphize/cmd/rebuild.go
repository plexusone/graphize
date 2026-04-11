package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/cache"
	"github.com/plexusone/graphize/pkg/extract"
	"github.com/plexusone/graphize/pkg/source"
	"github.com/spf13/cobra"
)

var (
	rebuildHTML      bool
	rebuildReport    bool
	rebuildTOON      bool
	rebuildSemantics string
	rebuildSkipMerge bool
)

var rebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Rebuild graph from sources and merge semantic edges",
	Long: `Rebuild the graph database from tracked sources.

This command is designed for the "clone and build" workflow:
1. Analyzes all tracked sources (AST extraction)
2. Merges semantic edges from agents/graph/semantic-edges.json (if present)
3. Optionally generates HTML visualization, report, or TOON export

Use this after cloning a repository that has checked-in semantic edges.

Examples:
  graphize rebuild                    # Analyze + merge semantic edges
  graphize rebuild --html             # Also generate HTML visualization
  graphize rebuild --report           # Also generate analysis report
  graphize rebuild --html --report    # Generate both
  graphize rebuild --skip-merge       # Skip merging semantic edges`,
	RunE: runRebuild,
}

func init() {
	rootCmd.AddCommand(rebuildCmd)
	rebuildCmd.Flags().BoolVar(&rebuildHTML, "html", false, "Generate HTML visualization after rebuild")
	rebuildCmd.Flags().BoolVar(&rebuildReport, "report", false, "Generate analysis report after rebuild")
	rebuildCmd.Flags().BoolVar(&rebuildTOON, "toon", false, "Generate TOON export after rebuild")
	rebuildCmd.Flags().StringVar(&rebuildSemantics, "semantics", "agents/graph/semantic-edges.json", "Path to semantic edges JSON file")
	rebuildCmd.Flags().BoolVar(&rebuildSkipMerge, "skip-merge", false, "Skip merging semantic edges")
}

func runRebuild(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Resolve graph path
	absGraphPath, err := filepath.Abs(graphPath)
	if err != nil {
		return fmt.Errorf("resolving graph path: %w", err)
	}

	// Check if initialized
	if _, err := os.Stat(absGraphPath); os.IsNotExist(err) {
		return fmt.Errorf("graph not initialized. Run 'graphize init' first")
	}

	// Load manifest
	manifest, err := source.LoadManifest(absGraphPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if len(manifest.Sources) == 0 {
		return fmt.Errorf("no sources tracked. Run 'graphize add <repo>' first")
	}

	fmt.Println("Graphize Rebuild")
	fmt.Println("================")
	fmt.Printf("Graph path: %s\n", absGraphPath)
	fmt.Printf("Sources: %d\n\n", len(manifest.Sources))

	// Step 1: Analyze (AST extraction)
	fmt.Println("Step 1: Analyzing sources (AST extraction)...")

	graphStore, err := store.NewFSStore(absGraphPath)
	if err != nil {
		return fmt.Errorf("creating graph store: %w", err)
	}

	extractor := extract.NewExtractor()
	c := cache.New(absGraphPath)
	extractor.WithCache(c)

	var totalNodes, totalEdges int
	for _, src := range manifest.Sources {
		g, stats := extractor.ExtractDirWithStats(src.Path)
		if err := graphStore.SaveGraph(g); err != nil {
			return fmt.Errorf("saving graph: %w", err)
		}
		nodeCount := len(g.Nodes)
		edgeCount := len(g.Edges)
		totalNodes += nodeCount
		totalEdges += edgeCount
		fmt.Printf("  %s: %d nodes, %d edges (cache: %d/%d hits)\n",
			filepath.Base(src.Path), nodeCount, edgeCount, stats.CacheHits, stats.TotalFiles)
	}
	fmt.Printf("  Total: %d nodes, %d edges\n\n", totalNodes, totalEdges)

	// Step 2: Merge semantic edges (if file exists and not skipped)
	if !rebuildSkipMerge {
		semanticsPath := rebuildSemantics
		if !filepath.IsAbs(semanticsPath) {
			cwd, _ := os.Getwd()
			semanticsPath = filepath.Join(cwd, semanticsPath)
		}

		if _, err := os.Stat(semanticsPath); err == nil {
			fmt.Printf("Step 2: Merging semantic edges from %s...\n", rebuildSemantics)

			if err := mergeSemanticEdges(absGraphPath, semanticsPath); err != nil {
				return fmt.Errorf("merge failed: %w", err)
			}
			fmt.Println()
		} else {
			fmt.Printf("Step 2: No semantic edges found at %s (skipping)\n\n", rebuildSemantics)
		}
	} else {
		fmt.Println("Step 2: Skipping semantic edge merge (--skip-merge)")
		fmt.Println()
	}

	// Step 3: Generate artifacts (optional)
	step := 3
	if rebuildHTML {
		fmt.Printf("Step %d: Generating HTML visualization...\n", step)
		exportOutput = "graph.html"
		if err := runExport(cmd, []string{"html"}); err != nil {
			return fmt.Errorf("HTML export failed: %w", err)
		}
		fmt.Println()
		step++
	}

	if rebuildReport {
		fmt.Printf("Step %d: Generating analysis report...\n", step)
		if err := runReport(cmd, args); err != nil {
			return fmt.Errorf("report generation failed: %w", err)
		}
		fmt.Println()
		step++
	}

	if rebuildTOON {
		fmt.Printf("Step %d: Generating TOON export...\n", step)
		toonOutput = "graph.toon"
		if err := runExportToon(cmd, args); err != nil {
			return fmt.Errorf("TOON export failed: %w", err)
		}
		fmt.Println()
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Rebuild complete in %s\n\n", elapsed.Round(time.Millisecond))
	fmt.Println("Next steps:")
	fmt.Println("  graphize query              # Query the graph")
	fmt.Println("  graphize report             # View analysis report")
	fmt.Println("  graphize export html        # Generate HTML visualization")

	return nil
}

// mergeSemanticEdges loads and merges semantic edges from a JSON file.
func mergeSemanticEdges(graphPath, semanticsPath string) error {
	// Read semantic edges file
	data, err := os.ReadFile(semanticsPath)
	if err != nil {
		return fmt.Errorf("reading semantic edges: %w", err)
	}

	// Parse
	semantic, err := extract.ParseSemanticJSON(data)
	if err != nil {
		return fmt.Errorf("parsing semantic edges: %w", err)
	}

	// Validate
	if err := extract.ValidateSemanticExtraction(semantic); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("  Parsed: %d nodes, %d edges\n", len(semantic.Nodes), len(semantic.Edges))

	if len(semantic.Edges) == 0 && len(semantic.Nodes) == 0 {
		fmt.Println("  Nothing to merge")
		return nil
	}

	// Load existing graph
	graphStore, err := store.NewFSStore(graphPath)
	if err != nil {
		return fmt.Errorf("opening graph store: %w", err)
	}

	existingNodes, err := graphStore.ListNodes()
	if err != nil {
		return fmt.Errorf("listing nodes: %w", err)
	}

	existingEdges, err := graphStore.ListEdges()
	if err != nil {
		return fmt.Errorf("listing edges: %w", err)
	}

	// Merge
	mergedNodes, mergedEdges := extract.MergeExtractions(existingNodes, existingEdges, semantic)

	newNodes := len(mergedNodes) - len(existingNodes)
	newEdges := len(mergedEdges) - len(existingEdges)

	fmt.Printf("  Merged: +%d nodes, +%d edges\n", newNodes, newEdges)

	// Save
	g := graph.NewGraph()
	for _, n := range mergedNodes {
		g.AddNode(n)
	}
	for _, e := range mergedEdges {
		g.AddEdge(e)
	}

	if err := graphStore.SaveGraph(g); err != nil {
		return fmt.Errorf("saving merged graph: %w", err)
	}

	return nil
}
