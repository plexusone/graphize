package htmlsite

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/pkg/source"
)

// ServiceMapping maps a service to its repository and local path.
type ServiceMapping struct {
	Name      string
	Slug      string
	RepoURL   string
	LocalPath string
}

// BuildServiceMappings extracts service-to-repo mappings from graph data.
// It finds svc:X nodes and their links_to repo:URL edges, then matches
// repo URLs to manifest source paths.
func BuildServiceMappings(nodes []*graph.Node, edges []*graph.Edge, manifest *source.Manifest) []ServiceMapping {
	var mappings []ServiceMapping

	// Find service nodes
	serviceNodes := make(map[string]*graph.Node) // node ID -> node
	for _, n := range nodes {
		if n.Type == "service" || strings.HasPrefix(n.ID, "svc:") {
			serviceNodes[n.ID] = n
		}
	}

	// Build repo URL -> local path mapping from manifest
	repoToPath := make(map[string]string)
	if manifest != nil {
		for _, s := range manifest.Sources {
			// Try to extract a normalized repo URL from the path
			// The local path is the source of truth for filtering
			repoToPath[s.Path] = s.Path
		}
	}

	// Find links_to edges from services to repos
	for _, e := range edges {
		if e.Type != "links_to" {
			continue
		}

		svcNode, isSvc := serviceNodes[e.From]
		if !isSvc {
			continue
		}

		// Check if target is a repo node
		if !strings.HasPrefix(e.To, "repo:") {
			continue
		}

		repoURL := strings.TrimPrefix(e.To, "repo:")
		svcName := strings.TrimPrefix(svcNode.ID, "svc:")
		if svcNode.Label != "" {
			svcName = svcNode.Label
		}

		// Try to find matching local path in manifest
		localPath := findLocalPathForRepo(repoURL, manifest)

		mapping := ServiceMapping{
			Name:      svcName,
			Slug:      slugify(svcName),
			RepoURL:   repoURL,
			LocalPath: localPath,
		}
		mappings = append(mappings, mapping)
	}

	// If no service mappings found but we have sources in manifest,
	// create a mapping for each source as a "service"
	if len(mappings) == 0 && manifest != nil && len(manifest.Sources) > 0 {
		for _, s := range manifest.Sources {
			name := filepath.Base(s.Path)
			mappings = append(mappings, ServiceMapping{
				Name:      name,
				Slug:      slugify(name),
				RepoURL:   "", // No repo URL known
				LocalPath: s.Path,
			})
		}
	}

	return mappings
}

// findLocalPathForRepo attempts to match a repo URL to a manifest source path.
func findLocalPathForRepo(repoURL string, manifest *source.Manifest) string {
	if manifest == nil {
		return ""
	}

	// Parse the repo URL to get the repo name
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return ""
	}

	// Extract repo name from URL path (e.g., "github.com/owner/repo" -> "repo")
	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(pathParts) == 0 {
		return ""
	}
	repoName := pathParts[len(pathParts)-1]
	repoName = strings.TrimSuffix(repoName, ".git")

	// Try to find a source path that contains this repo name
	for _, s := range manifest.Sources {
		if strings.HasSuffix(s.Path, "/"+repoName) || filepath.Base(s.Path) == repoName {
			return s.Path
		}
	}

	return ""
}

// FilterGraphByPath returns nodes and edges that belong to a specific local path.
// It filters nodes based on the source_file attribute prefix.
func FilterGraphByPath(nodes []*graph.Node, edges []*graph.Edge, localPath string) ([]*graph.Node, []*graph.Edge) {
	if localPath == "" {
		return nodes, edges
	}

	// Normalize path for comparison
	localPath = filepath.Clean(localPath)
	if !strings.HasSuffix(localPath, "/") {
		localPath += "/"
	}

	// Filter nodes
	nodeIDs := make(map[string]bool)
	var filteredNodes []*graph.Node

	for _, n := range nodes {
		// Check if node belongs to this path via source_file attribute
		if sourceFile, ok := n.Attrs["source_file"]; ok {
			if strings.HasPrefix(sourceFile, localPath) || strings.HasPrefix(sourceFile, strings.TrimSuffix(localPath, "/")) {
				nodeIDs[n.ID] = true
				filteredNodes = append(filteredNodes, n)
				continue
			}
		}

		// Check if node ID contains the path
		if strings.Contains(n.ID, localPath) || strings.Contains(n.ID, strings.TrimSuffix(localPath, "/")) {
			nodeIDs[n.ID] = true
			filteredNodes = append(filteredNodes, n)
		}
	}

	// Filter edges - only include edges where both endpoints are in filtered nodes
	var filteredEdges []*graph.Edge
	for _, e := range edges {
		if nodeIDs[e.From] && nodeIDs[e.To] {
			filteredEdges = append(filteredEdges, e)
		}
	}

	return filteredNodes, filteredEdges
}

// HasSystemNode checks if the graph contains a system node (indicating multi-service mode).
func HasSystemNode(nodes []*graph.Node) bool {
	for _, n := range nodes {
		if n.Type == "system" || strings.HasPrefix(n.ID, "system:") {
			return true
		}
	}
	return false
}

// GetSystemNodes returns all system-level nodes (system, service, repository types).
func GetSystemNodes(nodes []*graph.Node, edges []*graph.Edge) ([]*graph.Node, []*graph.Edge) {
	systemTypes := map[string]bool{
		"system":     true,
		"service":    true,
		"repository": true,
	}

	nodeIDs := make(map[string]bool)
	var filteredNodes []*graph.Node

	for _, n := range nodes {
		if systemTypes[n.Type] || strings.HasPrefix(n.ID, "svc:") || strings.HasPrefix(n.ID, "repo:") || strings.HasPrefix(n.ID, "system:") {
			nodeIDs[n.ID] = true
			filteredNodes = append(filteredNodes, n)
		}
	}

	// Include edges between system-level nodes
	var filteredEdges []*graph.Edge
	for _, e := range edges {
		if nodeIDs[e.From] && nodeIDs[e.To] {
			filteredEdges = append(filteredEdges, e)
		}
	}

	return filteredNodes, filteredEdges
}

// CountNodeTypes returns a map of node type -> count.
func CountNodeTypes(nodes []*graph.Node) map[string]int {
	counts := make(map[string]int)
	for _, n := range nodes {
		counts[n.Type]++
	}
	return counts
}

// CountEdgeTypes returns a map of edge type -> count.
func CountEdgeTypes(edges []*graph.Edge) map[string]int {
	counts := make(map[string]int)
	for _, e := range edges {
		counts[e.Type]++
	}
	return counts
}

// slugify converts a string to a URL-friendly slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}
