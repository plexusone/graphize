package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/exporters/htmlsite"
	"github.com/plexusone/graphize/pkg/metrics"
	"github.com/plexusone/graphize/pkg/source"
	"github.com/spf13/cobra"
)

var exportHTMLSiteCmd = &cobra.Command{
	Use:   "htmlsite",
	Short: "Export graph as multi-page HTML documentation site",
	Long: `Export the graph as a multi-page HTML documentation site.

In multi-service mode (when system-spec is present):
  - Index page shows system topology with service overview
  - Service pages show per-service code graphs filtered by repository

In single-repo mode (no system-spec):
  - Index page shows the full code graph
  - Optional community detection pages

The generated site is fully self-contained with embedded CSS and uses
CDN-hosted Cytoscape.js for visualization.

Examples:
  graphize export htmlsite -o ./site
  graphize export htmlsite -o ./docs --dark
  graphize export htmlsite -o ./site --no-communities --title "My System"`,
	RunE: runExportHTMLSite,
}

var (
	htmlSiteOutput        string
	htmlSiteTitle         string
	htmlSiteDark          bool
	htmlSiteNoCommunities bool
	htmlSiteNoIaC         bool
)

func init() {
	exportCmd.AddCommand(exportHTMLSiteCmd)

	exportHTMLSiteCmd.Flags().StringVarP(&htmlSiteOutput, "output", "o", "", "Output directory (required)")
	exportHTMLSiteCmd.Flags().StringVarP(&htmlSiteTitle, "title", "t", "Code Graph", "Site title")
	exportHTMLSiteCmd.Flags().BoolVar(&htmlSiteDark, "dark", false, "Use dark mode theme")
	exportHTMLSiteCmd.Flags().BoolVar(&htmlSiteNoCommunities, "no-communities", false, "Skip community pages")
	exportHTMLSiteCmd.Flags().BoolVar(&htmlSiteNoIaC, "no-iac", false, "Exclude Helm, Terraform, and other IaC nodes")

	_ = exportHTMLSiteCmd.MarkFlagRequired("output")
}

func runExportHTMLSite(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("no nodes found. Run 'graphize analyze' first")
	}

	// Load manifest for service mappings
	manifest, err := source.LoadManifest(path)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// Create generator
	gen := htmlsite.NewGenerator()
	gen.Title = htmlSiteTitle
	gen.DarkMode = htmlSiteDark
	gen.IncludeCommunities = !htmlSiteNoCommunities
	gen.ExcludeIaC = htmlSiteNoIaC

	// Generate site content
	content, err := gen.Generate(nodes, edges, manifest)
	if err != nil {
		return fmt.Errorf("generating site: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(htmlSiteOutput, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Write index.html
	indexPath := filepath.Join(htmlSiteOutput, "index.html")
	if err := os.WriteFile(indexPath, content.Index, 0600); err != nil {
		return fmt.Errorf("writing index.html: %w", err)
	}

	// Write service pages
	if len(content.Services) > 0 {
		servicesDir := filepath.Join(htmlSiteOutput, "services")
		if err := os.MkdirAll(servicesDir, 0755); err != nil {
			return fmt.Errorf("creating services directory: %w", err)
		}

		for slug, html := range content.Services {
			svcDir := filepath.Join(servicesDir, slug)
			if err := os.MkdirAll(svcDir, 0755); err != nil {
				return fmt.Errorf("creating service directory %s: %w", slug, err)
			}
			svcPath := filepath.Join(svcDir, "index.html")
			if err := os.WriteFile(svcPath, html, 0600); err != nil {
				return fmt.Errorf("writing service page %s: %w", slug, err)
			}
		}
	}

	// Write community pages
	if len(content.Communities) > 0 {
		communitiesDir := filepath.Join(htmlSiteOutput, "communities")
		if err := os.MkdirAll(communitiesDir, 0755); err != nil {
			return fmt.Errorf("creating communities directory: %w", err)
		}

		for id, html := range content.Communities {
			commPath := filepath.Join(communitiesDir, fmt.Sprintf("%d.html", id))
			if err := os.WriteFile(commPath, html, 0600); err != nil {
				return fmt.Errorf("writing community page %d: %w", id, err)
			}
		}
	}

	// Calculate total size
	var totalSize int64
	totalSize += int64(len(content.Index))
	for _, html := range content.Services {
		totalSize += int64(len(html))
	}
	for _, html := range content.Communities {
		totalSize += int64(len(html))
	}

	// Report stats
	fmt.Fprintf(os.Stderr, "Exported HTML site to %s\n", htmlSiteOutput)
	fmt.Fprintf(os.Stderr, "  Nodes: %d\n", len(nodes))
	fmt.Fprintf(os.Stderr, "  Edges: %d\n", len(edges))
	fmt.Fprintf(os.Stderr, "  Pages: %d (1 index + %d services + %d communities)\n",
		1+len(content.Services)+len(content.Communities),
		len(content.Services),
		len(content.Communities))
	fmt.Fprintf(os.Stderr, "  Size: %s\n", metrics.FormatBytes(totalSize))

	return nil
}
