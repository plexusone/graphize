package htmlsite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/pkg/source"
)

// Generator creates multi-page HTML documentation sites.
type Generator struct {
	// Title is the site title.
	Title string

	// Description is an optional site description.
	Description string

	// DarkMode enables dark theme.
	DarkMode bool

	// IncludeCommunities enables community page generation.
	IncludeCommunities bool

	// ExcludeIaC filters out Helm, Terraform, and other IaC nodes.
	ExcludeIaC bool
}

// NewGenerator creates a Generator with default settings.
func NewGenerator() *Generator {
	return &Generator{
		Title:              "Code Graph",
		DarkMode:           false,
		IncludeCommunities: true,
		ExcludeIaC:         false,
	}
}

// iacNodeTypes are node types to filter when ExcludeIaC is true.
var iacNodeTypes = map[string]bool{
	"helm_chart":       true,
	"terraform_module": true,
	"helm":             true,
	"terraform":        true,
}

// iacEdgeTypes are edge types to filter when ExcludeIaC is true.
var iacEdgeTypes = map[string]bool{
	"deploys": true,
	"manages": true,
}

// filterIaC removes IaC nodes and their edges from the graph.
func filterIaC(nodes []*graph.Node, edges []*graph.Edge) ([]*graph.Node, []*graph.Edge) {
	// Filter nodes
	iacNodeIDs := make(map[string]bool)
	var filteredNodes []*graph.Node
	for _, n := range nodes {
		if iacNodeTypes[n.Type] || strings.HasPrefix(n.ID, "helm:") || strings.HasPrefix(n.ID, "terraform:") {
			iacNodeIDs[n.ID] = true
			continue
		}
		filteredNodes = append(filteredNodes, n)
	}

	// Filter edges - remove edges involving IaC nodes or IaC edge types
	var filteredEdges []*graph.Edge
	for _, e := range edges {
		if iacNodeIDs[e.From] || iacNodeIDs[e.To] || iacEdgeTypes[e.Type] {
			continue
		}
		filteredEdges = append(filteredEdges, e)
	}

	return filteredNodes, filteredEdges
}

// SiteContent holds all generated pages.
type SiteContent struct {
	Index       []byte            // index.html content
	Services    map[string][]byte // service slug -> HTML content
	Communities map[int][]byte    // community ID -> HTML content
}

// ServiceInfo contains metadata about a service for display.
type ServiceInfo struct {
	Name        string
	Slug        string
	Description string
	RepoURL     string
	LocalPath   string
	NodeCount   int
	EdgeCount   int
}

// CommunityInfo contains metadata about a community for display.
type CommunityInfo struct {
	ID        int
	NodeCount int
	EdgeCount int
}

// IndexData is the template data for the index page.
type IndexData struct {
	Title          string
	Description    string
	DarkMode       bool
	CSS            string
	TotalNodes     int
	TotalEdges     int
	SourceCount    int
	HasServices    bool
	HasCommunities bool
	Services       []ServiceInfo
	Communities    []CommunityInfo
	GraphJSON      template.JS
}

// ServiceData is the template data for service pages.
type ServiceData struct {
	SiteTitle      string
	DarkMode       bool
	CSS            string
	HasCommunities bool
	Service        ServiceInfo
	NodeTypeCounts map[string]int
	EdgeTypeCounts map[string]int
	GraphJSON      template.JS
}

// Generate creates the multi-page HTML site.
func (g *Generator) Generate(nodes []*graph.Node, edges []*graph.Edge, manifest *source.Manifest) (*SiteContent, error) {
	// Filter out IaC nodes if requested
	if g.ExcludeIaC {
		nodes, edges = filterIaC(nodes, edges)
	}

	templates, err := LoadTemplates()
	if err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	content := &SiteContent{
		Services:    make(map[string][]byte),
		Communities: make(map[int][]byte),
	}

	// Check if we're in multi-service mode
	isMultiService := HasSystemNode(nodes)

	var services []ServiceInfo
	var serviceMappings []ServiceMapping

	if isMultiService {
		// Multi-service mode: extract service mappings
		serviceMappings = BuildServiceMappings(nodes, edges, manifest)

		for _, mapping := range serviceMappings {
			// Filter graph for this service
			svcNodes, svcEdges := FilterGraphByPath(nodes, edges, mapping.LocalPath)

			info := ServiceInfo{
				Name:      mapping.Name,
				Slug:      mapping.Slug,
				RepoURL:   mapping.RepoURL,
				LocalPath: mapping.LocalPath,
				NodeCount: len(svcNodes),
				EdgeCount: len(svcEdges),
			}
			services = append(services, info)

			// Generate service page if we have nodes
			if len(svcNodes) > 0 {
				html, err := g.generateServicePage(templates, info, svcNodes, svcEdges)
				if err != nil {
					return nil, fmt.Errorf("generating service page for %s: %w", mapping.Name, err)
				}
				content.Services[mapping.Slug] = html
			}
		}
	}

	// Generate index page
	var indexNodes, indexEdges []*graph.Node
	var indexEdgesTyped []*graph.Edge

	if isMultiService {
		// In multi-service mode, show only system topology on index
		indexNodes, indexEdgesTyped = GetSystemNodes(nodes, edges)
	} else {
		// Single repo mode: show all nodes
		indexNodes = nodes
		indexEdgesTyped = edges
	}
	indexEdges = indexNodes // for type counts (not used directly)
	_ = indexEdges

	indexHTML, err := g.generateIndexPage(templates, indexNodes, indexEdgesTyped, services, manifest)
	if err != nil {
		return nil, fmt.Errorf("generating index page: %w", err)
	}
	content.Index = indexHTML

	return content, nil
}

// generateIndexPage creates the index.html content.
func (g *Generator) generateIndexPage(templates *Templates, nodes []*graph.Node, edges []*graph.Edge, services []ServiceInfo, manifest *source.Manifest) ([]byte, error) {
	// Convert graph to Cytoscape JSON
	graphJSON, err := nodesToCytoscapeJSON(nodes, edges)
	if err != nil {
		return nil, fmt.Errorf("converting graph to JSON: %w", err)
	}

	sourceCount := 0
	if manifest != nil {
		sourceCount = len(manifest.Sources)
	}

	// Calculate total nodes/edges
	totalNodes := len(nodes)
	totalEdges := len(edges)

	// If we have services, count their totals instead
	if len(services) > 0 {
		totalNodes = 0
		totalEdges = 0
		for _, svc := range services {
			totalNodes += svc.NodeCount
			totalEdges += svc.EdgeCount
		}
	}

	data := IndexData{
		Title:          g.Title,
		Description:    g.Description,
		DarkMode:       g.DarkMode,
		CSS:            templates.CSS,
		TotalNodes:     totalNodes,
		TotalEdges:     totalEdges,
		SourceCount:    sourceCount,
		HasServices:    len(services) > 0,
		HasCommunities: g.IncludeCommunities && len(services) == 0, // Only show communities in single-repo mode
		Services:       services,
		Communities:    nil,                    // TODO: implement community detection
		GraphJSON:      template.JS(graphJSON), //nolint:gosec // G203: graphJSON is generated from trusted graph data, not user input
	}

	var buf bytes.Buffer
	if err := templates.Index.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing index template: %w", err)
	}

	return buf.Bytes(), nil
}

// generateServicePage creates a service page.
func (g *Generator) generateServicePage(templates *Templates, service ServiceInfo, nodes []*graph.Node, edges []*graph.Edge) ([]byte, error) {
	// Convert graph to Cytoscape JSON
	graphJSON, err := nodesToCytoscapeJSON(nodes, edges)
	if err != nil {
		return nil, fmt.Errorf("converting graph to JSON: %w", err)
	}

	data := ServiceData{
		SiteTitle:      g.Title,
		DarkMode:       g.DarkMode,
		CSS:            templates.CSS,
		HasCommunities: false, // Communities not supported in service pages yet
		Service:        service,
		NodeTypeCounts: CountNodeTypes(nodes),
		EdgeTypeCounts: CountEdgeTypes(edges),
		GraphJSON:      template.JS(graphJSON), //nolint:gosec // G203: graphJSON is generated from trusted graph data, not user input
	}

	var buf bytes.Buffer
	if err := templates.Service.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing service template: %w", err)
	}

	return buf.Bytes(), nil
}

// cytoscapeElement represents a Cytoscape.js element.
type cytoscapeElement struct {
	Data  map[string]interface{} `json:"data"`
	Group string                 `json:"group,omitempty"`
}

// nodesToCytoscapeJSON converts nodes and edges to Cytoscape.js JSON format.
func nodesToCytoscapeJSON(nodes []*graph.Node, edges []*graph.Edge) (string, error) {
	var elements []cytoscapeElement

	// Build node ID set
	nodeIDs := make(map[string]bool)
	for _, n := range nodes {
		nodeIDs[n.ID] = true
	}

	// Find missing node IDs referenced by edges and create placeholders
	missingNodes := make(map[string]bool)
	for _, e := range edges {
		if !nodeIDs[e.From] {
			missingNodes[e.From] = true
		}
		if !nodeIDs[e.To] {
			missingNodes[e.To] = true
		}
	}

	// Add existing nodes
	for _, n := range nodes {
		label := n.Label
		if label == "" {
			// Fallback to ID without prefix for cleaner display
			label = n.ID
			if idx := strings.Index(label, ":"); idx != -1 {
				label = label[idx+1:]
			}
		}

		data := map[string]interface{}{
			"id":    n.ID,
			"label": label,
			"type":  n.Type,
		}
		// Add extra attributes
		for k, v := range n.Attrs {
			data[k] = v
		}
		elements = append(elements, cytoscapeElement{
			Data:  data,
			Group: "nodes",
		})
	}

	// Add placeholder nodes for missing endpoints
	for nodeID := range missingNodes {
		// Derive label and type from node ID prefix
		label := nodeID
		nodeType := "unknown"
		if idx := strings.Index(nodeID, ":"); idx != -1 {
			prefix := nodeID[:idx]
			label = nodeID[idx+1:]
			// Map common prefixes to types
			switch prefix {
			case "svc":
				nodeType = "service"
			case "helm":
				nodeType = "helm_chart"
			case "terraform":
				nodeType = "terraform_module"
			case "repo":
				nodeType = "repository"
			case "rds", "dynamodb", "cloudsql":
				nodeType = "database"
			case "sqs", "sns", "pubsub":
				nodeType = "queue"
			case "s3", "gcs", "r2":
				nodeType = "storage"
			}
		}

		elements = append(elements, cytoscapeElement{
			Data: map[string]interface{}{
				"id":          nodeID,
				"label":       label,
				"type":        nodeType,
				"placeholder": true, // Mark as auto-generated
			},
			Group: "nodes",
		})
	}

	// Add all edges (no filtering needed - all endpoints now exist)
	for i, e := range edges {
		data := map[string]interface{}{
			"id":     fmt.Sprintf("e%d", i),
			"source": e.From,
			"target": e.To,
			"type":   e.Type,
		}

		if e.Confidence != "" {
			data["confidence"] = string(e.Confidence)
			if e.ConfidenceScore > 0 {
				data["confidence_score"] = e.ConfidenceScore
			}
		}
		// Add extra attributes
		for k, v := range e.Attrs {
			data[k] = v
		}
		elements = append(elements, cytoscapeElement{
			Data:  data,
			Group: "edges",
		})
	}

	// Validate all nodes have labels (fail fast instead of browser errors)
	var nodesWithoutLabels []string
	for _, elem := range elements {
		if elem.Group == "nodes" {
			label, hasLabel := elem.Data["label"]
			if !hasLabel {
				if id, ok := elem.Data["id"].(string); ok {
					nodesWithoutLabels = append(nodesWithoutLabels, id)
				}
			} else if labelStr, ok := label.(string); ok && labelStr == "" {
				if id, ok := elem.Data["id"].(string); ok {
					nodesWithoutLabels = append(nodesWithoutLabels, id)
				}
			}
		}
	}
	if len(nodesWithoutLabels) > 0 {
		return "", fmt.Errorf("nodes missing labels (would cause Cytoscape.js errors): %v", nodesWithoutLabels)
	}

	jsonBytes, err := json.Marshal(elements)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
