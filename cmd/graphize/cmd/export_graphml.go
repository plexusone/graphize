package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/spf13/cobra"
	"github.com/yaricom/goGraphML/graphml"
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

	// Create GraphML document
	gml := graphml.NewGraphML("graphize export")

	// Register node data keys
	nodeTypeKey, err := gml.RegisterKey(graphml.KeyForNode, "type", "Node type", reflect.String, "")
	if err != nil {
		return fmt.Errorf("registering node type key: %w", err)
	}

	nodeLabelKey, err := gml.RegisterKey(graphml.KeyForNode, "label", "Node label", reflect.String, "")
	if err != nil {
		return fmt.Errorf("registering node label key: %w", err)
	}

	nodePackageKey, err := gml.RegisterKey(graphml.KeyForNode, "package", "Package name", reflect.String, "")
	if err != nil {
		return fmt.Errorf("registering node package key: %w", err)
	}

	nodeSourceFileKey, err := gml.RegisterKey(graphml.KeyForNode, "source_file", "Source file", reflect.String, "")
	if err != nil {
		return fmt.Errorf("registering node source_file key: %w", err)
	}

	// Register edge data keys
	edgeTypeKey, err := gml.RegisterKey(graphml.KeyForEdge, "type", "Edge type", reflect.String, "")
	if err != nil {
		return fmt.Errorf("registering edge type key: %w", err)
	}

	edgeConfidenceKey, err := gml.RegisterKey(graphml.KeyForEdge, "confidence", "Edge confidence", reflect.String, "")
	if err != nil {
		return fmt.Errorf("registering edge confidence key: %w", err)
	}

	edgeConfidenceScoreKey, err := gml.RegisterKey(graphml.KeyForEdge, "confidence_score", "Confidence score", reflect.Float64, 0.0)
	if err != nil {
		return fmt.Errorf("registering edge confidence_score key: %w", err)
	}

	// Create graph
	edgeDirection := graphml.EdgeDirectionDirected
	if !graphmlDirected {
		edgeDirection = graphml.EdgeDirectionUndirected
	}

	g, err := gml.AddGraph("code-graph", edgeDirection, nil)
	if err != nil {
		return fmt.Errorf("creating graph: %w", err)
	}

	// Add nodes
	nodeMap := make(map[string]*graphml.Node)
	for _, n := range nodes {
		attrs := make(map[string]interface{})
		attrs[nodeTypeKey.Name] = n.Type
		attrs[nodeLabelKey.Name] = n.Label

		if n.Attrs != nil {
			if pkg := n.Attrs["package"]; pkg != "" {
				attrs[nodePackageKey.Name] = pkg
			}
			if sf := n.Attrs["source_file"]; sf != "" {
				attrs[nodeSourceFileKey.Name] = sf
			}
		}

		gmlNode, err := g.AddNode(attrs, n.ID)
		if err != nil {
			return fmt.Errorf("adding node %s: %w", n.ID, err)
		}
		nodeMap[n.ID] = gmlNode
	}

	// Add edges
	for _, e := range edges {
		fromNode := nodeMap[e.From]
		toNode := nodeMap[e.To]

		if fromNode == nil || toNode == nil {
			// Skip edges with missing nodes
			continue
		}

		attrs := make(map[string]interface{})
		attrs[edgeTypeKey.Name] = e.Type

		if e.Confidence != "" {
			attrs[edgeConfidenceKey.Name] = string(e.Confidence)
		}
		if e.ConfidenceScore > 0 {
			attrs[edgeConfidenceScoreKey.Name] = e.ConfidenceScore
		}

		edgeDesc := fmt.Sprintf("%s->%s", e.From, e.To)
		_, err := g.AddEdge(fromNode, toNode, attrs, graphml.EdgeDirectionDefault, edgeDesc)
		if err != nil {
			return fmt.Errorf("adding edge %s: %w", edgeDesc, err)
		}
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
	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := gml.Encode(f, true); err != nil {
		return fmt.Errorf("encoding GraphML: %w", err)
	}

	// Report stats
	fi, _ := os.Stat(output)
	fmt.Printf("Exported graph to %s\n", output)
	fmt.Printf("  Nodes: %d\n", len(nodes))
	fmt.Printf("  Edges: %d\n", len(edges))
	if fi != nil {
		fmt.Printf("  Size: %s\n", formatGraphMLSize(fi.Size()))
	}

	return nil
}

func formatGraphMLSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Ensure we're using the graph package (for documentation)
var _ = graph.NodeTypeFunction
