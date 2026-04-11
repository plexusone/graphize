package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/extract"
	"github.com/spf13/cobra"
)

var (
	mergeInput    string
	mergeValidate bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge semantic edges from LLM extraction into the graph",
	Long: `Merge LLM-extracted semantic edges into the existing AST graph.

The input file should contain JSON with semantic edges in the format:
{
  "nodes": [],
  "edges": [
    {
      "from": "node_id",
      "to": "node_id",
      "type": "inferred_depends",
      "confidence": "INFERRED",
      "confidence_score": 0.75,
      "reason": "explanation"
    }
  ]
}

Use --validate to check the input without modifying the graph.

Examples:
  graphize merge -i semantic-edges.json
  graphize merge -i semantic-edges.json --validate`,
	RunE: runMerge,
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().StringVarP(&mergeInput, "input", "i", "", "Input JSON file with semantic edges (required)")
	mergeCmd.Flags().BoolVar(&mergeValidate, "validate", false, "Validate input without merging")
	mergeCmd.MarkFlagRequired("input")
}

func runMerge(cmd *cobra.Command, args []string) error {
	// Read input file
	data, err := os.ReadFile(mergeInput)
	if err != nil {
		return fmt.Errorf("reading input file: %w", err)
	}

	// Parse semantic extraction
	semantic, err := extract.ParseSemanticJSON(data)
	if err != nil {
		return fmt.Errorf("parsing input: %w", err)
	}

	// Validate
	if err := extract.ValidateSemanticExtraction(semantic); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("Parsed semantic extraction:\n")
	fmt.Printf("  Nodes: %d\n", len(semantic.Nodes))
	fmt.Printf("  Edges: %d\n", len(semantic.Edges))

	// Show edge breakdown
	byType := make(map[string]int)
	byConf := make(map[string]int)
	for _, e := range semantic.Edges {
		byType[e.Type]++
		byConf[e.Confidence]++
	}

	if len(byType) > 0 {
		fmt.Printf("\n  Edge types:\n")
		for t, count := range byType {
			fmt.Printf("    %s: %d\n", t, count)
		}
	}

	if len(byConf) > 0 {
		fmt.Printf("\n  Confidence:\n")
		for c, count := range byConf {
			fmt.Printf("    %s: %d\n", c, count)
		}
	}

	if mergeValidate {
		fmt.Println("\nValidation passed. Use without --validate to merge.")
		return nil
	}

	// Load existing graph
	absGraphPath, err := filepath.Abs(graphPath)
	if err != nil {
		return fmt.Errorf("resolving graph path: %w", err)
	}

	graphStore, err := store.NewFSStore(absGraphPath)
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

	fmt.Printf("\nExisting graph:\n")
	fmt.Printf("  Nodes: %d\n", len(existingNodes))
	fmt.Printf("  Edges: %d\n", len(existingEdges))

	// Merge
	mergedNodes, mergedEdges := extract.MergeExtractions(existingNodes, existingEdges, semantic)

	newNodes := len(mergedNodes) - len(existingNodes)
	newEdges := len(mergedEdges) - len(existingEdges)

	fmt.Printf("\nAfter merge:\n")
	fmt.Printf("  Nodes: %d (+%d new)\n", len(mergedNodes), newNodes)
	fmt.Printf("  Edges: %d (+%d new)\n", len(mergedEdges), newEdges)

	// Save merged graph
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

	fmt.Printf("\nMerged graph saved to %s\n", absGraphPath)

	return nil
}

// SaveSemanticEdges saves semantic edges to a JSON file.
func SaveSemanticEdges(edges []extract.SemanticEdge, path string) error {
	data := extract.SemanticExtraction{
		Nodes: []extract.SemanticNode{},
		Edges: edges,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
