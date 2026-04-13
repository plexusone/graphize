package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/cache"
	"github.com/plexusone/graphize/pkg/extract"
	"github.com/plexusone/graphize/pkg/source"
	"github.com/spf13/cobra"
)

var (
	watchDebounce time.Duration
	watchHTML     bool
	watchReport   bool
	watchVerbose  bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for file changes and auto-rebuild graph",
	Long: `Monitor tracked source directories for file changes and automatically
rebuild the knowledge graph when Go files are modified.

Uses debouncing to batch rapid changes (default 500ms).

Examples:
  graphize watch                    # Watch and rebuild on changes
  graphize watch --html             # Also regenerate HTML visualization
  graphize watch --report           # Also regenerate analysis report
  graphize watch --debounce 1s      # Increase debounce delay`,
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().DurationVar(&watchDebounce, "debounce", 500*time.Millisecond, "Debounce delay for rapid changes")
	watchCmd.Flags().BoolVar(&watchHTML, "html", false, "Regenerate HTML visualization on changes")
	watchCmd.Flags().BoolVar(&watchReport, "report", false, "Regenerate analysis report on changes")
	watchCmd.Flags().BoolVar(&watchVerbose, "verbose", false, "Show detailed file change events")
}

func runWatch(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("no sources tracked. Use 'graphize add <path>' first")
	}

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	// Add source directories recursively
	for _, src := range manifest.Sources {
		if err := addWatchRecursive(watcher, src.Path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not watch %s: %v\n", src.Path, err)
		}
	}

	fmt.Printf("Watching %d source(s) for changes...\n", len(manifest.Sources))
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Debounce mechanism
	var (
		debounceTimer *time.Timer
		debounceMu    sync.Mutex
		pendingFiles  = make(map[string]bool)
	)

	rebuildGraph := func() {
		debounceMu.Lock()
		files := make([]string, 0, len(pendingFiles))
		for f := range pendingFiles {
			files = append(files, f)
		}
		pendingFiles = make(map[string]bool)
		debounceMu.Unlock()

		if len(files) == 0 {
			return
		}

		fmt.Printf("\n[%s] Rebuilding graph (%d file(s) changed)...\n",
			time.Now().Format("15:04:05"), len(files))

		startTime := time.Now()

		// Reload manifest in case it changed
		manifest, err := source.LoadManifest(absGraphPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading manifest: %v\n", err)
			return
		}

		// Create graph store
		graphStore, err := store.NewFSStore(absGraphPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening store: %v\n", err)
			return
		}

		// Create extractor with cache
		extractor := extract.NewMultiExtractor(extract.DefaultRegistry)
		c := cache.New(absGraphPath)
		extractor.WithCache(c)

		// Extract from each source
		var totalNodes, totalEdges int
		for _, src := range manifest.Sources {
			g, _ := extractor.ExtractDirWithStats(src.Path)
			if err := graphStore.SaveGraph(g); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving graph: %v\n", err)
				continue
			}
			totalNodes += g.NodeCount()
			totalEdges += g.EdgeCount()
		}

		fmt.Printf("  Extracted %d nodes, %d edges in %s\n",
			totalNodes, totalEdges, time.Since(startTime).Round(time.Millisecond))

		// Optionally regenerate HTML
		if watchHTML {
			htmlStart := time.Now()
			if err := generateWatchHTML(absGraphPath); err != nil {
				fmt.Fprintf(os.Stderr, "  Error generating HTML: %v\n", err)
			} else {
				fmt.Printf("  Generated graph.html in %s\n",
					time.Since(htmlStart).Round(time.Millisecond))
			}
		}

		// Optionally regenerate report
		if watchReport {
			reportStart := time.Now()
			if err := generateWatchReport(absGraphPath); err != nil {
				fmt.Fprintf(os.Stderr, "  Error generating report: %v\n", err)
			} else {
				fmt.Printf("  Generated GRAPH_REPORT.md in %s\n",
					time.Since(reportStart).Round(time.Millisecond))
			}
		}

		fmt.Printf("  Total time: %s\n", time.Since(startTime).Round(time.Millisecond))
	}

	// Event loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only care about Go files
			if !strings.HasSuffix(event.Name, ".go") {
				continue
			}

			// Skip test files for faster rebuilds
			if strings.HasSuffix(event.Name, "_test.go") {
				continue
			}

			// Only care about write/create/remove events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
				continue
			}

			if watchVerbose {
				fmt.Printf("  [%s] %s: %s\n",
					time.Now().Format("15:04:05"), event.Op, event.Name)
			}

			// Add to pending and reset debounce timer
			debounceMu.Lock()
			pendingFiles[event.Name] = true

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(watchDebounce, rebuildGraph)
			debounceMu.Unlock()

			// If a directory was created, add it to the watcher
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = addWatchRecursive(watcher, event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)

		case <-sigChan:
			fmt.Println("\nStopping watch...")
			return nil
		}
	}
}

// addWatchRecursive adds a directory and all subdirectories to the watcher.
func addWatchRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			return nil
		}

		// Skip hidden directories, vendor, node_modules
		name := info.Name()
		if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "testdata" {
			return filepath.SkipDir
		}

		return watcher.Add(path)
	})
}

// generateWatchHTML generates HTML visualization.
// TODO: Refactor export command to share this logic
func generateWatchHTML(graphPath string) error {
	// For now, skip HTML generation in watch mode
	// The user can run 'graphize export html' manually
	return nil
}

// generateWatchReport generates analysis report.
// TODO: Refactor report command to share this logic
func generateWatchReport(graphPath string) error {
	// For now, skip report generation in watch mode
	// The user can run 'graphize report' manually
	return nil
}
