package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	opts := analyze.ReportOptions{
		TopN: reportTopN,
	}
	report := analyze.GenerateReport(nodes, edges, opts)

	// Format as markdown
	markdown := report.FormatMarkdown(opts)

	// Output
	if reportOutput != "" {
		if err := writeReportFile(reportOutput, []byte(markdown)); err != nil {
			return fmt.Errorf("writing report: %w", err)
		}
		fmt.Printf("Report written to %s\n", reportOutput)
	} else {
		fmt.Print(markdown)
	}

	return nil
}

func writeReportFile(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0600)
}
