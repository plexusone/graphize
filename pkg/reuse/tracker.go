// Package reuse provides code reuse tracking and similarity detection.
package reuse

import (
	"fmt"
	"sort"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
)

// Tracker identifies reusable code patterns and similar structures.
type Tracker struct {
	nodes []*graph.Node
	edges []*graph.Edge
}

// NewTracker creates a new reuse tracker.
func NewTracker(nodes []*graph.Node, edges []*graph.Edge) *Tracker {
	return &Tracker{
		nodes: nodes,
		edges: edges,
	}
}

// ReuseReport contains the analysis results.
type ReuseReport struct {
	SimilarGroups      []SimilarGroup      `json:"similar_groups"`
	DuplicateNames     []DuplicateGroup    `json:"duplicate_names"`
	SharedDeps         []SharedDepGroup    `json:"shared_dependencies"`
	RefactorCandidates []RefactorCandidate `json:"refactor_candidates"`
	Summary            ReuseSummary        `json:"summary"`
}

// SimilarGroup represents nodes with similar signatures or structure.
type SimilarGroup struct {
	Pattern     string   `json:"pattern"`
	Description string   `json:"description"`
	Nodes       []string `json:"nodes"`
	Similarity  float64  `json:"similarity"`
	Type        string   `json:"type"`
}

// DuplicateGroup represents nodes with identical or near-identical names.
type DuplicateGroup struct {
	Name     string   `json:"name"`
	Nodes    []string `json:"nodes"`
	Packages []string `json:"packages"`
}

// SharedDepGroup represents nodes that share common dependencies.
type SharedDepGroup struct {
	SharedDeps []string `json:"shared_dependencies"`
	Nodes      []string `json:"nodes"`
	Count      int      `json:"dependency_count"`
}

// RefactorCandidate suggests potential refactoring opportunities.
type RefactorCandidate struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Nodes       []string `json:"nodes"`
	Suggestion  string   `json:"suggestion"`
	Priority    string   `json:"priority"` // high, medium, low
}

// ReuseSummary provides aggregate statistics.
type ReuseSummary struct {
	TotalNodes         int `json:"total_nodes"`
	SimilarGroupCount  int `json:"similar_group_count"`
	DuplicateCount     int `json:"duplicate_count"`
	RefactorCandidates int `json:"refactor_candidates"`
}

// Analyze performs the reuse analysis.
func (t *Tracker) Analyze() *ReuseReport {
	report := &ReuseReport{
		SimilarGroups:      []SimilarGroup{},
		DuplicateNames:     []DuplicateGroup{},
		SharedDeps:         []SharedDepGroup{},
		RefactorCandidates: []RefactorCandidate{},
	}

	// Find similar signatures
	report.SimilarGroups = append(report.SimilarGroups, t.findSimilarSignatures()...)

	// Find duplicate names across packages
	report.DuplicateNames = t.findDuplicateNames()

	// Find shared dependencies
	report.SharedDeps = t.findSharedDependencies()

	// Identify refactoring candidates
	report.RefactorCandidates = t.identifyRefactorCandidates(report)

	// Calculate summary
	report.Summary = ReuseSummary{
		TotalNodes:         len(t.nodes),
		SimilarGroupCount:  len(report.SimilarGroups),
		DuplicateCount:     len(report.DuplicateNames),
		RefactorCandidates: len(report.RefactorCandidates),
	}

	return report
}

// findSimilarSignatures groups functions with similar signatures.
func (t *Tracker) findSimilarSignatures() []SimilarGroup {
	var groups []SimilarGroup

	// Group by signature pattern
	sigPatterns := make(map[string][]string)

	for _, n := range t.nodes {
		if n.Type != "function" && n.Type != "method" {
			continue
		}

		sig := ""
		if n.Attrs != nil {
			sig = n.Attrs["signature"]
		}

		if sig == "" {
			continue
		}

		// Normalize signature to pattern
		pattern := normalizeSignature(sig)
		if pattern != "" {
			sigPatterns[pattern] = append(sigPatterns[pattern], n.ID)
		}
	}

	// Create groups for patterns with multiple matches
	for pattern, nodes := range sigPatterns {
		if len(nodes) >= 2 {
			groups = append(groups, SimilarGroup{
				Pattern:     pattern,
				Description: "Functions with similar signature pattern",
				Nodes:       nodes,
				Similarity:  1.0,
				Type:        "signature",
			})
		}
	}

	// Sort by group size
	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i].Nodes) > len(groups[j].Nodes)
	})

	// Limit to top 20
	if len(groups) > 20 {
		groups = groups[:20]
	}

	return groups
}

// normalizeSignature extracts a pattern from a function signature.
func normalizeSignature(sig string) string {
	// Extract parameter types pattern
	// e.g., "func foo(ctx context.Context, id string) error" -> "(Context,string)->error"

	sig = strings.TrimPrefix(sig, "func ")

	// Find parameters
	parenStart := strings.Index(sig, "(")
	parenEnd := strings.LastIndex(sig, ")")

	if parenStart < 0 || parenEnd < 0 || parenEnd <= parenStart {
		return ""
	}

	params := sig[parenStart+1 : parenEnd]
	returnPart := strings.TrimSpace(sig[parenEnd+1:])

	// Extract type names from params
	var paramTypes []string
	for _, p := range strings.Split(params, ",") {
		p = strings.TrimSpace(p)
		parts := strings.Fields(p)
		if len(parts) >= 2 {
			paramTypes = append(paramTypes, simplifyType(parts[len(parts)-1]))
		} else if len(parts) == 1 {
			paramTypes = append(paramTypes, simplifyType(parts[0]))
		}
	}

	// Simplify return type
	returnType := simplifyType(returnPart)

	if len(paramTypes) == 0 && returnType == "" {
		return ""
	}

	return "(" + strings.Join(paramTypes, ",") + ")->" + returnType
}

// simplifyType reduces a type to its base form.
func simplifyType(t string) string {
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(t, "*")
	t = strings.TrimPrefix(t, "[]")

	// Extract base type name
	if idx := strings.LastIndex(t, "."); idx >= 0 {
		t = t[idx+1:]
	}

	// Common type simplifications
	switch strings.ToLower(t) {
	case "context", "ctx":
		return "Context"
	case "error", "err":
		return "error"
	case "string", "str":
		return "string"
	case "int", "int64", "int32":
		return "int"
	case "bool":
		return "bool"
	case "":
		return ""
	default:
		return t
	}
}

// findDuplicateNames finds nodes with the same label in different packages.
func (t *Tracker) findDuplicateNames() []DuplicateGroup {
	nameMap := make(map[string][]struct {
		id  string
		pkg string
	})

	for _, n := range t.nodes {
		if n.Label == "" {
			continue
		}

		// Skip common names
		if isCommonName(n.Label) {
			continue
		}

		pkg := ""
		if n.Attrs != nil {
			pkg = n.Attrs["package"]
		}

		nameMap[n.Label] = append(nameMap[n.Label], struct {
			id  string
			pkg string
		}{n.ID, pkg})
	}

	var groups []DuplicateGroup
	for name, items := range nameMap {
		if len(items) < 2 {
			continue
		}

		// Check if they're in different packages
		pkgSet := make(map[string]bool)
		var nodes, pkgs []string
		for _, item := range items {
			nodes = append(nodes, item.id)
			if item.pkg != "" {
				pkgSet[item.pkg] = true
			}
		}

		for pkg := range pkgSet {
			pkgs = append(pkgs, pkg)
		}

		// Only report if in multiple packages
		if len(pkgs) >= 2 {
			groups = append(groups, DuplicateGroup{
				Name:     name,
				Nodes:    nodes,
				Packages: pkgs,
			})
		}
	}

	// Sort by node count
	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i].Nodes) > len(groups[j].Nodes)
	})

	// Limit to top 20
	if len(groups) > 20 {
		groups = groups[:20]
	}

	return groups
}

// isCommonName returns true for names that are expected to repeat.
func isCommonName(name string) bool {
	common := map[string]bool{
		"New": true, "Init": true, "Close": true, "Open": true,
		"Read": true, "Write": true, "String": true, "Error": true,
		"Get": true, "Set": true, "Delete": true, "Update": true,
		"Create": true, "List": true, "Find": true, "Search": true,
		"main": true, "init": true, "run": true, "start": true,
	}
	return common[name]
}

// findSharedDependencies finds nodes that depend on the same set of nodes.
func (t *Tracker) findSharedDependencies() []SharedDepGroup {
	// Build dependency map: node -> nodes it depends on
	deps := make(map[string][]string)
	for _, e := range t.edges {
		if e.Type == "calls" || e.Type == "imports" || e.Type == "uses" {
			deps[e.From] = append(deps[e.From], e.To)
		}
	}

	// Find nodes with identical dependency sets (3+ deps)
	depSigMap := make(map[string][]string)
	for node, nodeDeps := range deps {
		if len(nodeDeps) < 3 {
			continue
		}

		// Sort and create signature
		sorted := make([]string, len(nodeDeps))
		copy(sorted, nodeDeps)
		sort.Strings(sorted)
		sig := strings.Join(sorted, "|")

		depSigMap[sig] = append(depSigMap[sig], node)
	}

	var groups []SharedDepGroup
	for sig, nodes := range depSigMap {
		if len(nodes) < 2 {
			continue
		}

		sharedDeps := strings.Split(sig, "|")
		groups = append(groups, SharedDepGroup{
			SharedDeps: sharedDeps,
			Nodes:      nodes,
			Count:      len(sharedDeps),
		})
	}

	// Sort by shared dependency count
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Count > groups[j].Count
	})

	// Limit to top 10
	if len(groups) > 10 {
		groups = groups[:10]
	}

	return groups
}

// identifyRefactorCandidates suggests refactoring opportunities.
func (t *Tracker) identifyRefactorCandidates(report *ReuseReport) []RefactorCandidate {
	var candidates []RefactorCandidate

	// Large similar groups suggest extract common function
	for _, g := range report.SimilarGroups {
		if len(g.Nodes) >= 4 {
			candidates = append(candidates, RefactorCandidate{
				Type:        "extract_interface",
				Description: "Multiple functions with similar signatures",
				Nodes:       g.Nodes,
				Suggestion:  "Consider extracting a common interface or generic function",
				Priority:    "medium",
			})
		}
	}

	// Duplicate names in multiple packages
	for _, d := range report.DuplicateNames {
		if len(d.Packages) >= 3 {
			candidates = append(candidates, RefactorCandidate{
				Type:        "consolidate",
				Description: "Same name appears in " + strings.Join(d.Packages, ", "),
				Nodes:       d.Nodes,
				Suggestion:  "Consider consolidating into a shared package",
				Priority:    "low",
			})
		}
	}

	// Shared dependencies suggest extract service
	for _, s := range report.SharedDeps {
		if s.Count >= 5 && len(s.Nodes) >= 3 {
			candidates = append(candidates, RefactorCandidate{
				Type:        "extract_service",
				Description: fmt.Sprintf("Multiple nodes share %d dependencies", s.Count),
				Nodes:       s.Nodes,
				Suggestion:  "Consider extracting a service layer",
				Priority:    "high",
			})
		}
	}

	return candidates
}
