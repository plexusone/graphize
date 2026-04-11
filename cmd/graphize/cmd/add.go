package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/plexusone/graphize/pkg/source"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <repo-path>",
	Short: "Add a repository to track",
	Long: `Adds a git repository to the graph database for analysis.
Records the current commit hash and branch for currency tracking.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := args[0]

		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}

		// Resolve graph path
		absGraphPath, err := filepath.Abs(graphPath)
		if err != nil {
			return fmt.Errorf("resolving graph path: %w", err)
		}

		// Load existing manifest
		manifest, err := source.LoadManifest(absGraphPath)
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		// Check if already tracked
		existing := manifest.GetSource(absPath)
		isUpdate := existing != nil

		// Create source from path (gets current commit/branch)
		src, err := source.NewSourceFromPath(absPath)
		if err != nil {
			return fmt.Errorf("reading repository: %w", err)
		}

		// Add or update source in manifest
		manifest.AddSource(src)

		// Save manifest
		if err := source.SaveManifest(absGraphPath, manifest); err != nil {
			return fmt.Errorf("saving manifest: %w", err)
		}

		action := "added"
		if isUpdate {
			action = "updated"
		}

		result := map[string]any{
			"status": action,
			"source": map[string]any{
				"path":        src.Path,
				"commit":      src.Commit,
				"branch":      src.Branch,
				"analyzed_at": src.AnalyzedAt.Format("2006-01-02T15:04:05Z"),
			},
			"total_sources": len(manifest.Sources),
			"message":       fmt.Sprintf("Source %s. Use 'graphize analyze' to extract graph.", action),
		}
		return printOutput(result)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
