// Package cmd provides the CLI commands for graphize.
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/output"
	"github.com/spf13/cobra"
)

var (
	// outputFormat is the output format flag (toon, json, yaml).
	outputFormat string

	// graphPath is the path to the graph database.
	graphPath string
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "graphize",
	Short: "Turn codebases into queryable knowledge graphs",
	Long: `Graphize extracts structure from Go codebases and builds
queryable knowledge graphs stored in GraphFS format.

Output is TOON format by default (agent-friendly, token-efficient).
Use --format to change: json, yaml.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "toon",
		"Output format: toon (default), json, yaml")
	rootCmd.PersistentFlags().StringVarP(&graphPath, "graph", "g", ".graphize",
		"Path to the graph database directory")
}

// getFormatter returns a formatter based on the --format flag.
func getFormatter() (output.Formatter, error) {
	return output.NewFormatter(output.Format(outputFormat))
}

// printOutput formats and prints output.
func printOutput(v any) error {
	formatter, err := getFormatter()
	if err != nil {
		return err
	}
	data, err := formatter.Format(v)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// GraphData holds loaded graph nodes and edges.
type GraphData struct {
	Nodes []*graph.Node
	Edges []*graph.Edge
}

// loadGraph loads nodes and edges from the graph store.
func loadGraph() (*GraphData, error) {
	absGraphPath, err := filepath.Abs(graphPath)
	if err != nil {
		return nil, fmt.Errorf("resolving graph path: %w", err)
	}

	graphStore, err := store.NewFSStore(absGraphPath)
	if err != nil {
		return nil, fmt.Errorf("opening graph store: %w", err)
	}

	nodes, err := graphStore.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("loading nodes: %w", err)
	}

	edges, err := graphStore.ListEdges()
	if err != nil {
		return nil, fmt.Errorf("loading edges: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in graph at %s", absGraphPath)
	}

	return &GraphData{Nodes: nodes, Edges: edges}, nil
}
