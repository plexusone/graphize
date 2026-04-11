package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/plexusone/graphize/pkg/source"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of tracked sources",
	Long: `Shows the currency status of all tracked repositories.
Compares recorded commit hashes with current HEAD to detect staleness.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
				"graph_path": absGraphPath,
				"sources":    []any{},
				"message":    "No sources tracked. Use 'graphize add <repo>' to add sources.",
			}
			return printOutput(result)
		}

		// Check status of all sources
		statuses, err := manifest.CheckAllStatus()
		if err != nil {
			return fmt.Errorf("checking status: %w", err)
		}

		// Build output
		var sourcesOut []map[string]any
		staleCount := 0
		for _, status := range statuses {
			s := map[string]any{
				"path":            status.Source.Path,
				"tracked_commit":  status.Source.Commit,
				"tracked_branch":  status.Source.Branch,
				"analyzed_at":     status.Source.AnalyzedAt.Format("2006-01-02T15:04:05Z"),
				"current_commit":  status.CurrentCommit,
				"current_branch":  status.CurrentBranch,
				"is_stale":        status.IsStale,
				"commits_behind":  status.CommitsBehind,
			}
			sourcesOut = append(sourcesOut, s)
			if status.IsStale {
				staleCount++
			}
		}

		message := "All sources are current."
		if staleCount > 0 {
			message = fmt.Sprintf("%d source(s) are stale. Use 'graphize add <repo>' to update.", staleCount)
		}

		result := map[string]any{
			"graph_path":   absGraphPath,
			"total":        len(manifest.Sources),
			"stale":        staleCount,
			"sources":      sourcesOut,
			"message":      message,
		}
		return printOutput(result)
	},
}

var statusSourceCmd = &cobra.Command{
	Use:   "source <repo-path>",
	Short: "Check status of a specific source",
	Args:  cobra.ExactArgs(1),
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

		// Load manifest
		manifest, err := source.LoadManifest(absGraphPath)
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		// Check if tracked
		tracked := manifest.GetSource(absPath)
		if tracked == nil {
			// Not tracked, just show current state
			src, err := source.NewSourceFromPath(absPath)
			if err != nil {
				return fmt.Errorf("reading repository: %w", err)
			}

			result := map[string]any{
				"path":           src.Path,
				"current_commit": src.Commit,
				"current_branch": src.Branch,
				"tracked":        false,
				"message":        "Source not tracked. Use 'graphize add' to track.",
			}
			return printOutput(result)
		}

		// Get status
		status, err := source.CheckStatus(tracked)
		if err != nil {
			return fmt.Errorf("checking status: %w", err)
		}

		result := map[string]any{
			"path":           status.Source.Path,
			"tracked_commit": status.Source.Commit,
			"tracked_branch": status.Source.Branch,
			"analyzed_at":    status.Source.AnalyzedAt.Format("2006-01-02T15:04:05Z"),
			"current_commit": status.CurrentCommit,
			"current_branch": status.CurrentBranch,
			"tracked":        true,
			"is_stale":       status.IsStale,
			"commits_behind": status.CommitsBehind,
		}

		if status.IsStale {
			result["message"] = fmt.Sprintf("Source is %d commit(s) behind. Use 'graphize add' to update.", status.CommitsBehind)
		} else {
			result["message"] = "Source is current."
		}

		return printOutput(result)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.AddCommand(statusSourceCmd)
}
