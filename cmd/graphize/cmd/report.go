package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/analyze"
	"github.com/spf13/cobra"
)

var (
	reportTopN   int
	reportOutput string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate analysis report for the graph",
	Long: `Analyze the graph and generate a report with:
  - God nodes (most connected entities)
  - Community detection results
  - Surprising connections
  - Package statistics
  - Edge confidence breakdown

The report helps understand the architecture and identify potential issues.`,
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().IntVar(&reportTopN, "top", 10, "Number of top items to show")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "Output file (default: stdout)")
}

func runReport(cmd *cobra.Command, args []string) error {
	// Resolve graph path
	absGraphPath, err := filepath.Abs(graphPath)
	if err != nil {
		return fmt.Errorf("resolving graph path: %w", err)
	}

	// Load graph
	s, err := store.NewFSStore(absGraphPath)
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

	// Generate report
	var sb strings.Builder

	sb.WriteString("# Graph Analysis Report\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	nodesByType := analyze.NodesByType(nodes)
	edgesByType := analyze.EdgesByType(edges)
	edgesByConf := analyze.EdgesByConfidence(edges)

	sb.WriteString(fmt.Sprintf("- **Total Nodes:** %d\n", len(nodes)))
	sb.WriteString(fmt.Sprintf("- **Total Edges:** %d\n", len(edges)))
	sb.WriteString("\n")

	sb.WriteString("### Node Types\n\n")
	sb.WriteString("| Type | Count |\n")
	sb.WriteString("|------|-------|\n")
	for t, ns := range nodesByType {
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", t, len(ns)))
	}
	sb.WriteString("\n")

	sb.WriteString("### Edge Types\n\n")
	sb.WriteString("| Type | Count |\n")
	sb.WriteString("|------|-------|\n")
	for t, es := range edgesByType {
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", t, len(es)))
	}
	sb.WriteString("\n")

	sb.WriteString("### Edge Confidence\n\n")
	sb.WriteString("| Confidence | Count |\n")
	sb.WriteString("|------------|-------|\n")
	for c, es := range edgesByConf {
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", c, len(es)))
	}
	sb.WriteString("\n")

	// God Nodes
	sb.WriteString("## God Nodes (Most Connected)\n\n")
	sb.WriteString("These are the most connected entities - core architectural abstractions.\n\n")
	godNodes := analyze.GodNodes(nodes, edges, reportTopN)
	sb.WriteString("| Rank | Label | Type | In | Out | Total |\n")
	sb.WriteString("|------|-------|------|-----|-----|-------|\n")
	for i, g := range godNodes {
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %d | %d | %d |\n",
			i+1, g.Label, g.Type, g.InDegree, g.OutDegree, g.Total))
	}
	sb.WriteString("\n")

	// Community Detection
	sb.WriteString("## Communities\n\n")
	sb.WriteString("Groups of related code detected using the Louvain algorithm.\n\n")
	clusterResult := analyze.DetectCommunities(nodes, edges)
	communityLabels := analyze.CommunityLabels(clusterResult.Communities, nodes)

	if clusterResult.Modularity != 0 {
		sb.WriteString(fmt.Sprintf("**Modularity (Q):** %.4f\n\n", clusterResult.Modularity))
	}

	sb.WriteString("| ID | Size | Cohesion | Label |\n")
	sb.WriteString("|----|------|----------|-------|\n")
	for _, c := range clusterResult.Communities {
		label := communityLabels[c.ID]
		sb.WriteString(fmt.Sprintf("| %d | %d | %.2f | %s |\n",
			c.ID, c.Size, c.Cohesion, label))
	}
	sb.WriteString("\n")

	// Surprising Connections
	sb.WriteString("## Surprising Connections\n\n")
	sb.WriteString("Non-obvious relationships that may indicate architectural decisions or issues.\n\n")

	communityMap := make(map[int][]string)
	for _, c := range clusterResult.Communities {
		communityMap[c.ID] = c.Members
	}
	surprises := analyze.SurprisingConnections(nodes, edges, communityMap, reportTopN)

	if len(surprises) == 0 {
		sb.WriteString("No surprising connections found.\n\n")
	} else {
		sb.WriteString("| From | To | Type | Confidence | Why |\n")
		sb.WriteString("|------|-----|------|------------|-----|\n")
		for _, s := range surprises {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				s.FromLabel, s.ToLabel, s.Type, s.Confidence, s.Why))
		}
		sb.WriteString("\n")
	}

	// Isolated Nodes
	sb.WriteString("## Isolated Nodes\n\n")
	sb.WriteString("Nodes with very few connections - potential documentation gaps.\n\n")
	isolated := analyze.IsolatedNodes(nodes, edges, 1)
	if len(isolated) == 0 {
		sb.WriteString("No isolated nodes found.\n\n")
	} else {
		shown := isolated
		if len(shown) > reportTopN {
			shown = shown[:reportTopN]
		}
		sb.WriteString("| Label | Type |\n")
		sb.WriteString("|-------|------|\n")
		for _, n := range shown {
			label := n.Label
			if label == "" {
				label = n.ID
			}
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", label, n.Type))
		}
		if len(isolated) > reportTopN {
			sb.WriteString(fmt.Sprintf("\n*...and %d more isolated nodes*\n", len(isolated)-reportTopN))
		}
		sb.WriteString("\n")
	}

	// Package Statistics
	sb.WriteString("## Package Statistics\n\n")
	pkgStats := analyze.AnalyzePackages(nodes, edges)
	sb.WriteString("| Package | Files | Functions | Types | Imports |\n")
	sb.WriteString("|---------|-------|-----------|-------|----------|\n")
	for _, p := range pkgStats {
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d |\n",
			p.Name, p.Files, p.Functions, p.Types, p.Imports))
	}
	sb.WriteString("\n")

	// Cross-file edges
	crossFile := analyze.CrossFileEdges(nodes, edges)
	sb.WriteString("## Cross-File Dependencies\n\n")
	sb.WriteString(fmt.Sprintf("**%d edges** connect nodes across different source files.\n\n", len(crossFile)))

	// Suggested Questions
	sb.WriteString("## Suggested Questions\n\n")
	sb.WriteString("Questions to explore based on graph analysis.\n\n")
	questions := analyze.SuggestQuestions(nodes, edges, communityMap, 5)
	if len(questions) == 1 && questions[0].Type == "no_signal" {
		sb.WriteString(fmt.Sprintf("*%s*\n\n", questions[0].Why))
	} else {
		for i, q := range questions {
			if q.Question != "" {
				sb.WriteString(fmt.Sprintf("**%d. %s**\n\n", i+1, q.Question))
				sb.WriteString(fmt.Sprintf("   *%s*\n\n", q.Why))
			}
		}
	}

	// Output
	report := sb.String()
	if reportOutput != "" {
		if err := writeFile(reportOutput, []byte(report)); err != nil {
			return fmt.Errorf("writing report: %w", err)
		}
		fmt.Printf("Report written to %s\n", reportOutput)
	} else {
		fmt.Print(report)
	}

	return nil
}

func writeFile(path string, data []byte) error {
	return writeFileWithDir(path, data)
}

func writeFileWithDir(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0644)
}
