package analyze

import (
	"fmt"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
)

// Report contains all sections of a graph analysis report.
type Report struct {
	// Summary contains basic counts
	Summary ReportSummary `json:"summary"`

	// GodNodes are the most connected nodes
	GodNodes []GodNode `json:"god_nodes"`

	// Bridges are nodes with high betweenness centrality
	Bridges []BridgeNode `json:"bridges"`

	// Communities detected via Louvain algorithm
	Communities  []CommunityInfo  `json:"communities"`
	Modularity   float64          `json:"modularity"`
	CommunityMap map[int][]string `json:"-"` // for internal use

	// Surprises are unexpected connections
	Surprises []Surprise `json:"surprises"`

	// IsolatedNodes have very few connections
	IsolatedNodes []*graph.Node `json:"isolated_nodes"`

	// PackageStats per package
	PackageStats []PackageStats `json:"package_stats"`

	// CrossFileEdgeCount is the number of edges crossing file boundaries
	CrossFileEdgeCount int `json:"cross_file_edge_count"`

	// Questions suggested based on analysis
	Questions []Question `json:"questions"`
}

// ReportSummary contains basic graph statistics.
type ReportSummary struct {
	TotalNodes       int            `json:"total_nodes"`
	TotalEdges       int            `json:"total_edges"`
	NodeTypeCounts   map[string]int `json:"node_types"`
	EdgeTypeCounts   map[string]int `json:"edge_types"`
	ConfidenceCounts map[string]int `json:"confidence_counts"`
}

// CommunityInfo contains community details for the report.
type CommunityInfo struct {
	ID       int      `json:"id"`
	Size     int      `json:"size"`
	Cohesion float64  `json:"cohesion"`
	Label    string   `json:"label"`
	Members  []string `json:"members,omitempty"`
}

// ReportOptions configures report generation.
type ReportOptions struct {
	// TopN limits result lists (god nodes, surprises, etc.)
	TopN int

	// IncludeMembers includes community member lists in output
	IncludeMembers bool
}

// DefaultReportOptions returns sensible defaults.
func DefaultReportOptions() ReportOptions {
	return ReportOptions{
		TopN:           10,
		IncludeMembers: false,
	}
}

// GenerateReport creates a complete analysis report from the graph.
func GenerateReport(nodes []*graph.Node, edges []*graph.Edge, opts ReportOptions) *Report {
	report := &Report{}

	// Summary
	nodesByType := NodesByType(nodes)
	edgesByType := EdgesByType(edges)
	edgesByConf := EdgesByConfidence(edges)

	report.Summary = ReportSummary{
		TotalNodes:       len(nodes),
		TotalEdges:       len(edges),
		NodeTypeCounts:   make(map[string]int),
		EdgeTypeCounts:   make(map[string]int),
		ConfidenceCounts: make(map[string]int),
	}
	for t, ns := range nodesByType {
		report.Summary.NodeTypeCounts[t] = len(ns)
	}
	for t, es := range edgesByType {
		report.Summary.EdgeTypeCounts[t] = len(es)
	}
	for c, es := range edgesByConf {
		report.Summary.ConfidenceCounts[string(c)] = len(es)
	}

	// God Nodes
	report.GodNodes = GodNodes(nodes, edges, opts.TopN)

	// Bridges
	report.Bridges = FindBridges(nodes, edges, opts.TopN)

	// Communities
	clusterResult := DetectCommunities(nodes, edges)
	communityLabels := CommunityLabels(clusterResult.Communities, nodes)
	report.Modularity = clusterResult.Modularity
	report.CommunityMap = make(map[int][]string)

	for _, c := range clusterResult.Communities {
		info := CommunityInfo{
			ID:       c.ID,
			Size:     c.Size,
			Cohesion: c.Cohesion,
			Label:    communityLabels[c.ID],
		}
		if opts.IncludeMembers {
			info.Members = c.Members
		}
		report.Communities = append(report.Communities, info)
		report.CommunityMap[c.ID] = c.Members
	}

	// Surprising Connections
	report.Surprises = SurprisingConnections(nodes, edges, report.CommunityMap, opts.TopN)

	// Isolated Nodes
	report.IsolatedNodes = IsolatedNodes(nodes, edges, 1)

	// Package Statistics
	report.PackageStats = AnalyzePackages(nodes, edges)

	// Cross-file edges
	crossFile := CrossFileEdges(nodes, edges)
	report.CrossFileEdgeCount = len(crossFile)

	// Suggested Questions
	report.Questions = SuggestQuestions(nodes, edges, report.CommunityMap, 5)

	return report
}

// FormatMarkdown converts the report to markdown format.
func (r *Report) FormatMarkdown(opts ReportOptions) string {
	var sb strings.Builder

	sb.WriteString("# Graph Analysis Report\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	fmt.Fprintf(&sb, "- **Total Nodes:** %d\n", r.Summary.TotalNodes)
	fmt.Fprintf(&sb, "- **Total Edges:** %d\n", r.Summary.TotalEdges)
	sb.WriteString("\n")

	sb.WriteString("### Node Types\n\n")
	sb.WriteString("| Type | Count |\n")
	sb.WriteString("|------|-------|\n")
	for t, count := range r.Summary.NodeTypeCounts {
		fmt.Fprintf(&sb, "| %s | %d |\n", t, count)
	}
	sb.WriteString("\n")

	sb.WriteString("### Edge Types\n\n")
	sb.WriteString("| Type | Count |\n")
	sb.WriteString("|------|-------|\n")
	for t, count := range r.Summary.EdgeTypeCounts {
		fmt.Fprintf(&sb, "| %s | %d |\n", t, count)
	}
	sb.WriteString("\n")

	sb.WriteString("### Edge Confidence\n\n")
	sb.WriteString("| Confidence | Count |\n")
	sb.WriteString("|------------|-------|\n")
	for c, count := range r.Summary.ConfidenceCounts {
		fmt.Fprintf(&sb, "| %s | %d |\n", c, count)
	}
	sb.WriteString("\n")

	// God Nodes
	sb.WriteString("## God Nodes (Most Connected)\n\n")
	sb.WriteString("These are the most connected entities - core architectural abstractions.\n\n")
	sb.WriteString("| Rank | Label | Type | In | Out | Total |\n")
	sb.WriteString("|------|-------|------|-----|-----|-------|\n")
	for i, g := range r.GodNodes {
		fmt.Fprintf(&sb, "| %d | %s | %s | %d | %d | %d |\n",
			i+1, g.Label, g.Type, g.InDegree, g.OutDegree, g.Total)
	}
	sb.WriteString("\n")

	// Bridges
	sb.WriteString("## Bridges (Betweenness Centrality)\n\n")
	sb.WriteString("Nodes that connect different parts of the codebase - architectural chokepoints.\n\n")
	if len(r.Bridges) == 0 {
		sb.WriteString("No significant bridges found.\n\n")
	} else {
		sb.WriteString("| Rank | Label | Type | Centrality |\n")
		sb.WriteString("|------|-------|------|------------|\n")
		for i, b := range r.Bridges {
			fmt.Fprintf(&sb, "| %d | %s | %s | %.2f |\n",
				i+1, b.Label, b.Type, b.Centrality)
		}
		sb.WriteString("\n")
	}

	// Communities
	sb.WriteString("## Communities\n\n")
	sb.WriteString("Groups of related code detected using the Louvain algorithm.\n\n")
	if r.Modularity != 0 {
		fmt.Fprintf(&sb, "**Modularity (Q):** %.4f\n\n", r.Modularity)
	}
	sb.WriteString("| ID | Size | Cohesion | Label |\n")
	sb.WriteString("|----|------|----------|-------|\n")
	for _, c := range r.Communities {
		fmt.Fprintf(&sb, "| %d | %d | %.2f | %s |\n",
			c.ID, c.Size, c.Cohesion, c.Label)
	}
	sb.WriteString("\n")

	// Surprising Connections
	sb.WriteString("## Surprising Connections\n\n")
	sb.WriteString("Non-obvious relationships that may indicate architectural decisions or issues.\n\n")
	if len(r.Surprises) == 0 {
		sb.WriteString("No surprising connections found.\n\n")
	} else {
		sb.WriteString("| From | To | Type | Confidence | Why |\n")
		sb.WriteString("|------|-----|------|------------|-----|\n")
		for _, s := range r.Surprises {
			fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s |\n",
				s.FromLabel, s.ToLabel, s.Type, s.Confidence, s.Why)
		}
		sb.WriteString("\n")
	}

	// Isolated Nodes
	sb.WriteString("## Isolated Nodes\n\n")
	sb.WriteString("Nodes with very few connections - potential documentation gaps.\n\n")
	if len(r.IsolatedNodes) == 0 {
		sb.WriteString("No isolated nodes found.\n\n")
	} else {
		shown := r.IsolatedNodes
		if len(shown) > opts.TopN {
			shown = shown[:opts.TopN]
		}
		sb.WriteString("| Label | Type |\n")
		sb.WriteString("|-------|------|\n")
		for _, n := range shown {
			label := n.Label
			if label == "" {
				label = n.ID
			}
			fmt.Fprintf(&sb, "| %s | %s |\n", label, n.Type)
		}
		if len(r.IsolatedNodes) > opts.TopN {
			fmt.Fprintf(&sb, "\n*...and %d more isolated nodes*\n", len(r.IsolatedNodes)-opts.TopN)
		}
		sb.WriteString("\n")
	}

	// Package Statistics
	sb.WriteString("## Package Statistics\n\n")
	sb.WriteString("| Package | Files | Functions | Types | Imports |\n")
	sb.WriteString("|---------|-------|-----------|-------|----------|\n")
	for _, p := range r.PackageStats {
		fmt.Fprintf(&sb, "| %s | %d | %d | %d | %d |\n",
			p.Name, p.Files, p.Functions, p.Types, p.Imports)
	}
	sb.WriteString("\n")

	// Cross-file Dependencies
	sb.WriteString("## Cross-File Dependencies\n\n")
	fmt.Fprintf(&sb, "**%d edges** connect nodes across different source files.\n\n", r.CrossFileEdgeCount)

	// Suggested Questions
	sb.WriteString("## Suggested Questions\n\n")
	sb.WriteString("Questions to explore based on graph analysis.\n\n")
	if len(r.Questions) == 1 && r.Questions[0].Type == "no_signal" {
		fmt.Fprintf(&sb, "*%s*\n\n", r.Questions[0].Why)
	} else {
		for i, q := range r.Questions {
			if q.Question != "" {
				fmt.Fprintf(&sb, "**%d. %s**\n\n", i+1, q.Question)
				fmt.Fprintf(&sb, "   *%s*\n\n", q.Why)
			}
		}
	}

	return sb.String()
}
