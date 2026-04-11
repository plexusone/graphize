package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cytoscape "github.com/grokify/cytoscape-go"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export [format]",
	Short: "Export graph to various formats",
	Long: `Export the graph database to various formats.

Supported formats:
  html     - Interactive Cytoscape.js visualization (default)
  json     - Cytoscape.js JSON format
  toon     - TOON format (agent-optimized, token-efficient)
  graphml  - GraphML XML format (for Gephi, yEd, Cytoscape)
  cypher   - Neo4j Cypher CREATE statements

Examples:
  graphize export html -o graph.html
  graphize export json -o graph.json
  graphize export graphml -o graph.graphml
  graphize export cypher -o graph.cypher
  graphize export toon -o graph.toon.gz --gzip
  graphize export html --title "My Project" --dark`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExport,
}

var (
	exportOutput      string
	exportTitle       string
	exportDescription string
	exportDarkMode    bool
	exportNoSearch    bool
	exportNoFilters   bool
	exportNoLegend    bool
	exportNoStats     bool
	exportNoExport    bool
	exportNoLayout    bool
	exportMinimap     bool
	exportLayout      string
)

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path")
	exportCmd.Flags().StringVarP(&exportTitle, "title", "t", "Code Graph", "Graph title")
	exportCmd.Flags().StringVarP(&exportDescription, "description", "d", "", "Graph description")
	exportCmd.Flags().BoolVar(&exportDarkMode, "dark", false, "Use dark mode theme")
	exportCmd.Flags().BoolVar(&exportNoSearch, "no-search", false, "Disable search box")
	exportCmd.Flags().BoolVar(&exportNoFilters, "no-filters", false, "Disable type filters")
	exportCmd.Flags().BoolVar(&exportNoLegend, "no-legend", false, "Disable legend")
	exportCmd.Flags().BoolVar(&exportNoStats, "no-stats", false, "Disable statistics display")
	exportCmd.Flags().BoolVar(&exportNoExport, "no-export", false, "Disable export buttons")
	exportCmd.Flags().BoolVar(&exportNoLayout, "no-layout", false, "Disable layout selector")
	exportCmd.Flags().BoolVar(&exportMinimap, "minimap", false, "Enable minimap")
	exportCmd.Flags().StringVar(&exportLayout, "layout", "dagre", "Initial layout (cose, dagre, cola, circle, grid)")
}

func runExport(cmd *cobra.Command, args []string) error {
	format := "html"
	if len(args) > 0 {
		format = strings.ToLower(args[0])
	}

	// Determine graph path
	path := graphPath
	if path == "" {
		path = ".graphize"
	}

	// Load graph from store
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
		return fmt.Errorf("no nodes found in graph database. Run 'graphize analyze' first")
	}

	// Build cytoscape graph
	g := cytoscape.NewGraph()
	g.SetTitle(exportTitle)
	g.SetStyle(cytoscape.CodeGraphStyle())

	// Set layout
	switch exportLayout {
	case "cose":
		g.SetLayout(&cytoscape.CoseLayout{Animate: true})
	case "dagre":
		g.SetLayout(&cytoscape.DagreLayout{RankDir: "TB", NodeSep: 60, RankSep: 100})
	case "cola":
		g.SetLayout(&cytoscape.ColaLayout{Animate: true})
	case "circle":
		g.SetLayout(&cytoscape.CircleLayout{})
	case "grid":
		g.SetLayout(&cytoscape.GridLayout{})
	default:
		g.SetLayout(&cytoscape.DagreLayout{RankDir: "TB", NodeSep: 60, RankSep: 100})
	}

	// Convert nodes
	for _, n := range nodes {
		node := cytoscape.NodeWithType(n.ID, n.Label, n.Type)
		if n.Attrs != nil {
			for k, v := range n.Attrs {
				node.SetExtra(k, v)
			}
		}
		g.AddNode(node)
	}

	// Convert edges
	for _, e := range edges {
		edgeID := fmt.Sprintf("%s->%s", e.From, e.To)
		edge := cytoscape.EdgeWithType(edgeID, e.From, e.To, "", e.Type)
		// Include confidence for edge coloring in visualization
		if e.Confidence != "" {
			edge.SetExtra("confidence", string(e.Confidence))
			if e.ConfidenceScore > 0 {
				edge.SetExtra("confidence_score", fmt.Sprintf("%.2f", e.ConfidenceScore))
			}
		}
		g.AddEdge(edge)
	}

	// Generate output
	switch format {
	case "html":
		return exportHTML(g)
	case "json":
		return exportJSON(g)
	default:
		return fmt.Errorf("unsupported format: %s (use html or json)", format)
	}
}

func exportHTML(g *cytoscape.Graph) error {
	opts := cytoscape.HTMLOptions{
		Title:              exportTitle,
		Description:        exportDescription,
		ShowSearch:         !exportNoSearch,
		ShowFilters:        !exportNoFilters,
		ShowLegend:         !exportNoLegend,
		ShowStats:          !exportNoStats,
		ShowExport:         !exportNoExport,
		ShowLayoutSelector: !exportNoLayout,
		ShowMinimap:        exportMinimap,
		DarkMode:           exportDarkMode,
		MaxLabelLength:     40,
		UseDagre:           true,
		UseCola:            true,
	}

	html, err := g.ToHTML(opts)
	if err != nil {
		return fmt.Errorf("generating HTML: %w", err)
	}

	// Determine output path
	output := exportOutput
	if output == "" {
		output = "graph.html"
	}

	// Ensure directory exists
	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	if err := os.WriteFile(output, html, 0644); err != nil {
		return fmt.Errorf("writing HTML file: %w", err)
	}

	fmt.Printf("Exported graph to %s\n", output)
	fmt.Printf("  Nodes: %d\n", g.Metadata.NodeCount)
	fmt.Printf("  Edges: %d\n", g.Metadata.EdgeCount)

	return nil
}

func exportJSON(g *cytoscape.Graph) error {
	data, err := g.ToJSON()
	if err != nil {
		return fmt.Errorf("generating JSON: %w", err)
	}

	// Determine output path
	output := exportOutput
	if output == "" {
		output = "graph.json"
	}

	// Ensure directory exists
	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		return fmt.Errorf("writing JSON file: %w", err)
	}

	fmt.Printf("Exported graph to %s\n", output)
	fmt.Printf("  Nodes: %d\n", g.Metadata.NodeCount)
	fmt.Printf("  Edges: %d\n", g.Metadata.EdgeCount)

	return nil
}
