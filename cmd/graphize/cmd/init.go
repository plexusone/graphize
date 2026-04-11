package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new graphize database",
	Long:  `Creates a new GraphFS-backed graph database in the specified directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		absPath, err := filepath.Abs(graphPath)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}

		// Check if already exists
		if _, err := os.Stat(absPath); err == nil {
			return fmt.Errorf("graph database already exists at %s", absPath)
		}

		// Create the store (which creates directory structure)
		_, err = store.NewFSStore(absPath)
		if err != nil {
			return fmt.Errorf("creating graph store: %w", err)
		}

		result := map[string]any{
			"status":  "initialized",
			"path":    absPath,
			"message": "Graph database created. Use 'graphize add <repo>' to add sources.",
		}
		return printOutput(result)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
