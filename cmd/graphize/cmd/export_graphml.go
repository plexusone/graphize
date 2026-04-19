package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/exporters/graphml"
	"github.com/plexusone/graphize/pkg/metrics"
	"github.com/spf13/cobra"
)

var exportGraphMLCmd = &cobra.Command{
	Use:   "graphml",
	Short: "Export graph in GraphML format",
	Long: `Export the graph in GraphML format for use with graph visualization tools.

GraphML is an XML-based format supported by:
  - Gephi (gephi.org)
  - yEd (yworks.com/yed)
  - Cytoscape (cytoscape.org)
  - NetworkX (Python)

The export includes:
  - All nodes with type and label attributes
  - All edges with type and confidence attributes
  - Custom data keys for node/edge properties

Examples:
  graphize export graphml -o graph.graphml
  graphize export graphml --directed  # Force directed edges`,
	RunE: runExportGraphML,
}

var (
	graphmlOutput   string
	graphmlDirected bool
)

func init() {
	exportCmd.AddCommand(exportGraphMLCmd)
	exportGraphMLCmd.Flags().StringVarP(&graphmlOutput, "output", "o", "", "Output file path")
	exportGraphMLCmd.Flags().BoolVar(&graphmlDirected, "directed", true, "Export as directed graph (default: true)")
}

func runExportGraphML(cmd *cobra.Command, args []string) error {
	path := graphPath
	if path == "" {
		path = ".graphize"
	}

	s, err := store.NewFSStore(path)
	if err != nil {
		return fmt.Errorf("opening graph store: %w", err)
	}

	nodes, err := s.ListNodes()
	if err != nil {
		return fmt.Errorf("listing nodes: %w", err)
	}

	edges, err := s.ListEdges()
	if err != nil {
		return fmt.Errorf("listing edges: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found. Run 'graphize analyze' first")
	}

	// Generate GraphML
	gen := graphml.NewGenerator()
	gen.Directed = graphmlDirected

	result, err := gen.Generate(nodes, edges)
	if err != nil {
		return fmt.Errorf("generating GraphML: %w", err)
	}

	// Determine output path
	output := graphmlOutput
	if output == "" {
		output = "graph.graphml"
	}

	// Ensure directory exists
	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	// Write GraphML file
	if err := os.WriteFile(output, result.Data, 0600); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	// Report stats
	fmt.Printf("Exported graph to %s\n", output)
	fmt.Printf("  Nodes: %d\n", result.NodeCount)
	fmt.Printf("  Edges: %d\n", result.EdgeCount)
	if result.SkippedEdges > 0 {
		fmt.Printf("  Skipped edges: %d\n", result.SkippedEdges)
	}
	fmt.Printf("  Size: %s\n", metrics.FormatBytes(int64(len(result.Data))))

	return nil
}
