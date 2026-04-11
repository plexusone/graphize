package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/output"
	"github.com/spf13/cobra"
)

var initAgentsCmd = &cobra.Command{
	Use:   "init-agents",
	Short: "Initialize agent framework directories",
	Long: `Creates agents/ directory structure with graph exports for AI agent consumption.

The command creates:
  agents/
  ├── graph/
  │   ├── GRAPH_SUMMARY.md    # Markdown summary (checkable)
  │   └── GRAPH.toon.gz       # Compressed TOON export (checkable)
  ├── specs/                   # multi-agent-spec definitions
  │   └── .gitkeep
  ├── plugins/                 # Generated plugins (claude, kiro, codex, gemini)
  │   └── .gitkeep
  └── .gitignore              # Ignore local-only files

Examples:
  graphize init-agents
  graphize init-agents --dir .agents
  graphize init-agents --force
  graphize init-agents --no-export`,
	RunE: runInitAgents,
}

var (
	agentsDir      string
	agentsForce    bool
	agentsNoExport bool
)

func init() {
	rootCmd.AddCommand(initAgentsCmd)
	initAgentsCmd.Flags().StringVarP(&agentsDir, "dir", "d", "agents", "Directory name (default: agents)")
	initAgentsCmd.Flags().BoolVar(&agentsForce, "force", false, "Overwrite existing graph exports")
	initAgentsCmd.Flags().BoolVar(&agentsNoExport, "no-export", false, "Create structure only, skip graph exports")
}

func runInitAgents(cmd *cobra.Command, args []string) error {
	path := graphPath
	if path == "" {
		path = ".graphize"
	}

	// Verify graph database exists (only if we need to export)
	var nodes, edges int
	if !agentsNoExport {
		s, err := store.NewFSStore(path)
		if err != nil {
			return fmt.Errorf("opening graph store: %w\nRun 'graphize init' and 'graphize analyze' first", err)
		}

		nodeList, err := s.ListNodes()
		if err != nil {
			return fmt.Errorf("listing nodes: %w", err)
		}

		edgeList, err := s.ListEdges()
		if err != nil {
			return fmt.Errorf("listing edges: %w", err)
		}

		nodes = len(nodeList)
		edges = len(edgeList)

		if nodes == 0 {
			fmt.Println("Warning: Graph is empty. Run 'graphize analyze' to populate it.")
		}

		// Generate exports if we have data or force is set
		if nodes > 0 || agentsForce {
			if err := generateAgentExports(nodeList, edgeList, agentsForce); err != nil {
				return err
			}
		}
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(agentsDir, "graph"),
		filepath.Join(agentsDir, "specs"),
		filepath.Join(agentsDir, "plugins"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Create .gitkeep files in empty directories
	gitkeepDirs := []string{
		filepath.Join(agentsDir, "specs"),
		filepath.Join(agentsDir, "plugins"),
	}

	for _, dir := range gitkeepDirs {
		gitkeepPath := filepath.Join(dir, ".gitkeep")
		if _, err := os.Stat(gitkeepPath); os.IsNotExist(err) {
			if err := os.WriteFile(gitkeepPath, []byte{}, 0644); err != nil {
				return fmt.Errorf("creating .gitkeep in %s: %w", dir, err)
			}
		}
	}

	// Create .gitignore
	if err := createAgentsGitignore(); err != nil {
		return err
	}

	// Print summary
	fmt.Printf("Initialized agent framework: %s/\n", agentsDir)
	fmt.Printf("  Created directories: graph/, specs/, plugins/\n")
	if !agentsNoExport && nodes > 0 {
		fmt.Printf("  Generated exports:\n")
		fmt.Printf("    - graph/GRAPH_SUMMARY.md\n")
		fmt.Printf("    - graph/GRAPH.toon.gz\n")
		fmt.Printf("  Graph stats: %d nodes, %d edges\n", nodes, edges)
	} else if agentsNoExport {
		fmt.Printf("  Skipped graph exports (--no-export)\n")
	} else {
		fmt.Printf("  No graph exports (empty graph)\n")
	}

	return nil
}

func generateAgentExports(nodeList []*graph.Node, edgeList []*graph.Edge, force bool) error {
	graphDir := filepath.Join(agentsDir, "graph")

	// Ensure graph directory exists
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		return fmt.Errorf("creating graph directory: %w", err)
	}

	// Generate GRAPH_SUMMARY.md
	summaryPath := filepath.Join(graphDir, "GRAPH_SUMMARY.md")
	if force || !fileExists(summaryPath) {
		markdown := output.GenerateSummaryMarkdown(nodeList, edgeList, output.SummaryOptions{
			TopN: 10,
		})
		if err := os.WriteFile(summaryPath, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("writing GRAPH_SUMMARY.md: %w", err)
		}
	}

	// Generate GRAPH.toon.gz
	toonPath := filepath.Join(graphDir, "GRAPH.toon.gz")
	if force || !fileExists(toonPath) {
		content := output.GenerateTOON(nodeList, edgeList, output.TOONOptions{
			NoExtra: false,
			Compact: false,
		})
		if err := output.WriteTOONGzipped(toonPath, content); err != nil {
			return fmt.Errorf("writing GRAPH.toon.gz: %w", err)
		}
	}

	return nil
}

func createAgentsGitignore() error {
	gitignorePath := filepath.Join(agentsDir, ".gitignore")

	content := `# Local agent artifacts (not for version control)
# Graph exports are regenerated via: graphize init-agents

# Large uncompressed exports
graph/*.toon
graph/*.json

# Plugin build artifacts
plugins/**/build/
plugins/**/dist/

# Keep structure
!.gitkeep
`

	if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
