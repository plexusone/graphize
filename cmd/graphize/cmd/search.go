package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/search"
	"github.com/spf13/cobra"
)

var (
	searchLimit   int
	searchOffset  int
	searchTypes   string
	searchFuzzy   int
	searchReindex bool
	searchStats   bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search over the knowledge graph",
	Long: `Full-text search over nodes in the knowledge graph.

Searches across node labels, docstrings, packages, and signatures.
The search index is automatically built and updated.

Examples:
  graphize search "authentication"           # Search for authentication-related code
  graphize search "HTTP handler" --type function  # Search only functions
  graphize search "config" --fuzzy 1         # Fuzzy search (typo tolerant)
  graphize search --stats                    # Show index statistics
  graphize search --reindex                  # Rebuild the search index

Search fields:
  - label: Function/class/variable names
  - doc: Docstrings and comments
  - package: Package names
  - signature: Function signatures
  - source_file: File paths
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve graph path
		absGraphPath, err := filepath.Abs(graphPath)
		if err != nil {
			return fmt.Errorf("resolving graph path: %w", err)
		}

		// Create searcher
		searcher, err := search.NewSearcher(absGraphPath)
		if err != nil {
			return fmt.Errorf("creating searcher: %w", err)
		}
		defer searcher.Close()

		// Handle stats request
		if searchStats {
			return showSearchStats(searcher)
		}

		// Handle reindex request
		if searchReindex {
			return reindexGraph(absGraphPath, searcher)
		}

		// Require query for search
		if len(args) == 0 {
			// If no args and no flags, show stats
			return showSearchStats(searcher)
		}

		// Check if index is empty, auto-index if needed
		stats, err := searcher.Stats()
		if err == nil && stats.TotalDocs == 0 {
			fmt.Println("Index is empty, building index...")
			if err := reindexGraph(absGraphPath, searcher); err != nil {
				return err
			}
		}

		// Perform search
		query := strings.Join(args, " ")
		opts := search.SearchOptions{
			Limit:     searchLimit,
			Offset:    searchOffset,
			FuzzyDist: searchFuzzy,
			Highlight: true,
		}

		if searchTypes != "" {
			opts.NodeTypes = strings.Split(searchTypes, ",")
		}

		output, err := searcher.Search(query, opts)
		if err != nil {
			return fmt.Errorf("searching: %w", err)
		}

		// Add index stats to output
		output.IndexStats, _ = searcher.Stats()

		return printOutput(output)
	},
}

func showSearchStats(searcher *search.Searcher) error {
	stats, err := searcher.Stats()
	if err != nil {
		return fmt.Errorf("getting stats: %w", err)
	}

	output := map[string]any{
		"index_path":   stats.IndexPath,
		"total_docs":   stats.TotalDocs,
		"storage_size": formatBytes(stats.StorageSize),
		"message":      "Use 'graphize search <query>' to search the graph.",
	}

	return printOutput(output)
}

func reindexGraph(graphPath string, searcher *search.Searcher) error {
	// Load graph
	graphStore, err := store.NewFSStore(graphPath)
	if err != nil {
		return fmt.Errorf("opening graph store: %w", err)
	}

	nodes, err := graphStore.ListNodes()
	if err != nil {
		return fmt.Errorf("loading nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found in graph at %s", graphPath)
	}

	// Reindex
	if err := searcher.Reindex(nodes); err != nil {
		return fmt.Errorf("reindexing: %w", err)
	}

	fmt.Printf("Indexed %d nodes\n", len(nodes))
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "Maximum number of results")
	searchCmd.Flags().IntVar(&searchOffset, "offset", 0, "Skip first N results")
	searchCmd.Flags().StringVar(&searchTypes, "type", "", "Filter by node type(s), comma-separated (function,class,struct)")
	searchCmd.Flags().IntVar(&searchFuzzy, "fuzzy", 0, "Fuzzy matching edit distance (0=exact, 1-2=fuzzy)")
	searchCmd.Flags().BoolVar(&searchReindex, "reindex", false, "Rebuild the search index")
	searchCmd.Flags().BoolVar(&searchStats, "stats", false, "Show index statistics")
}
