// Package graphml provides GraphML format export for code graphs.
package graphml

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/yaricom/goGraphML/graphml"
)

// Generator creates GraphML output from a code graph.
type Generator struct {
	// Directed controls whether edges are directed (default: true).
	Directed bool

	// GraphID is the identifier for the graph element.
	GraphID string

	// Description is the GraphML document description.
	Description string
}

// NewGenerator creates a Generator with default settings.
func NewGenerator() *Generator {
	return &Generator{
		Directed:    true,
		GraphID:     "code-graph",
		Description: "graphize export",
	}
}

// Result contains the generated GraphML and statistics.
type Result struct {
	// Data is the generated GraphML content.
	Data []byte

	// NodeCount is the number of nodes exported.
	NodeCount int

	// EdgeCount is the number of edges exported.
	EdgeCount int

	// SkippedEdges is the number of edges skipped due to missing nodes.
	SkippedEdges int
}

// Generate creates GraphML output from nodes and edges.
func (g *Generator) Generate(nodes []*graph.Node, edges []*graph.Edge) (*Result, error) {
	var buf bytes.Buffer
	result, err := g.WriteTo(&buf, nodes, edges)
	if err != nil {
		return nil, err
	}
	result.Data = buf.Bytes()
	return result, nil
}

// WriteTo writes GraphML output to the provided writer.
func (g *Generator) WriteTo(w io.Writer, nodes []*graph.Node, edges []*graph.Edge) (*Result, error) {
	result := &Result{
		NodeCount: len(nodes),
	}

	// Create GraphML document
	gml := graphml.NewGraphML(g.Description)

	// Register node data keys
	nodeTypeKey, err := gml.RegisterKey(graphml.KeyForNode, "type", "Node type", reflect.String, "")
	if err != nil {
		return nil, fmt.Errorf("registering node type key: %w", err)
	}

	nodeLabelKey, err := gml.RegisterKey(graphml.KeyForNode, "label", "Node label", reflect.String, "")
	if err != nil {
		return nil, fmt.Errorf("registering node label key: %w", err)
	}

	nodePackageKey, err := gml.RegisterKey(graphml.KeyForNode, "package", "Package name", reflect.String, "")
	if err != nil {
		return nil, fmt.Errorf("registering node package key: %w", err)
	}

	nodeSourceFileKey, err := gml.RegisterKey(graphml.KeyForNode, "source_file", "Source file", reflect.String, "")
	if err != nil {
		return nil, fmt.Errorf("registering node source_file key: %w", err)
	}

	// Register edge data keys
	edgeTypeKey, err := gml.RegisterKey(graphml.KeyForEdge, "type", "Edge type", reflect.String, "")
	if err != nil {
		return nil, fmt.Errorf("registering edge type key: %w", err)
	}

	edgeConfidenceKey, err := gml.RegisterKey(graphml.KeyForEdge, "confidence", "Edge confidence", reflect.String, "")
	if err != nil {
		return nil, fmt.Errorf("registering edge confidence key: %w", err)
	}

	edgeConfidenceScoreKey, err := gml.RegisterKey(graphml.KeyForEdge, "confidence_score", "Confidence score", reflect.Float64, 0.0)
	if err != nil {
		return nil, fmt.Errorf("registering edge confidence_score key: %w", err)
	}

	// Create graph element
	edgeDirection := graphml.EdgeDirectionDirected
	if !g.Directed {
		edgeDirection = graphml.EdgeDirectionUndirected
	}

	gr, err := gml.AddGraph(g.GraphID, edgeDirection, nil)
	if err != nil {
		return nil, fmt.Errorf("creating graph: %w", err)
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

		gmlNode, err := gr.AddNode(attrs, n.ID)
		if err != nil {
			return nil, fmt.Errorf("adding node %s: %w", n.ID, err)
		}
		nodeMap[n.ID] = gmlNode
	}

	// Add edges
	edgeCount := 0
	skippedEdges := 0
	for _, e := range edges {
		fromNode := nodeMap[e.From]
		toNode := nodeMap[e.To]

		if fromNode == nil || toNode == nil {
			skippedEdges++
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
		_, err := gr.AddEdge(fromNode, toNode, attrs, graphml.EdgeDirectionDefault, edgeDesc)
		if err != nil {
			return nil, fmt.Errorf("adding edge %s: %w", edgeDesc, err)
		}
		edgeCount++
	}

	result.EdgeCount = edgeCount
	result.SkippedEdges = skippedEdges

	// Encode to writer
	if err := gml.Encode(w, true); err != nil {
		return nil, fmt.Errorf("encoding GraphML: %w", err)
	}

	return result, nil
}
