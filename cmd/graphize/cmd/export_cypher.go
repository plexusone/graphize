package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/exporters/cypher"
	"github.com/plexusone/graphize/pkg/metrics"
	"github.com/spf13/cobra"
)

var exportCypherCmd = &cobra.Command{
	Use:   "cypher",
	Short: "Export graph as Neo4j Cypher statements",
	Long: `Export the graph as Neo4j Cypher CREATE statements.

The output can be used to import the graph into Neo4j:
  - Copy/paste into Neo4j Browser
  - Use cypher-shell: cat graph.cypher | cypher-shell -u neo4j -p password
  - Use neo4j-admin import

Examples:
  graphize export cypher -o graph.cypher
  graphize export cypher | cypher-shell -u neo4j -p secret`,
	RunE: runExportCypher,
}

var cypherOutput string

func init() {
	exportCmd.AddCommand(exportCypherCmd)
	exportCypherCmd.Flags().StringVarP(&cypherOutput, "output", "o", "", "Output file path (stdout if not specified)")
}

func runExportCypher(cmd *cobra.Command, args []string) error {
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

	// Generate Cypher using the library
	gen := cypher.NewGenerator()
	output := gen.Generate(nodes, edges)

	// Output
	if cypherOutput == "" {
		// Write to stdout
		fmt.Print(output)
		return nil
	}

	// Write to file
	if dir := filepath.Dir(cypherOutput); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	if err := os.WriteFile(cypherOutput, []byte(output), 0600); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	// Report stats
	fi, _ := os.Stat(cypherOutput)
	fmt.Fprintf(os.Stderr, "Exported graph to %s\n", cypherOutput)
	fmt.Fprintf(os.Stderr, "  Nodes: %d\n", len(nodes))
	fmt.Fprintf(os.Stderr, "  Edges: %d\n", len(edges))
	if fi != nil {
		fmt.Fprintf(os.Stderr, "  Size: %s\n", metrics.FormatBytes(fi.Size()))
	}

	return nil
}
