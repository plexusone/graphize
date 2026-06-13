// Package patterns provides architectural pattern detection in code graphs.
package patterns

import (
	"sort"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
)

// Detector identifies common architectural patterns in a code graph.
type Detector struct {
	nodes    []*graph.Node
	edges    []*graph.Edge
	nodeMap  map[string]*graph.Node
	outEdges map[string][]*graph.Edge // from -> edges
	inEdges  map[string][]*graph.Edge // to -> edges
}

// NewDetector creates a new pattern detector.
func NewDetector(nodes []*graph.Node, edges []*graph.Edge) *Detector {
	d := &Detector{
		nodes:    nodes,
		edges:    edges,
		nodeMap:  make(map[string]*graph.Node),
		outEdges: make(map[string][]*graph.Edge),
		inEdges:  make(map[string][]*graph.Edge),
	}

	for _, n := range nodes {
		d.nodeMap[n.ID] = n
	}

	for _, e := range edges {
		d.outEdges[e.From] = append(d.outEdges[e.From], e)
		d.inEdges[e.To] = append(d.inEdges[e.To], e)
	}

	return d
}

// PatternReport contains all detected patterns.
type PatternReport struct {
	Architectural []ArchPattern   `json:"architectural"`
	Structural    []StructPattern `json:"structural"`
	AntiPatterns  []AntiPattern   `json:"anti_patterns"`
	Summary       PatternSummary  `json:"summary"`
}

// ArchPattern represents an architectural pattern instance.
type ArchPattern struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Nodes       []string `json:"nodes"`
	Confidence  float64  `json:"confidence"`
	Location    string   `json:"location,omitempty"`
}

// StructPattern represents a structural graph pattern.
type StructPattern struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Nodes       []string `json:"nodes"`
	Edges       int      `json:"edge_count"`
}

// AntiPattern represents a detected anti-pattern.
type AntiPattern struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"` // high, medium, low
	Description string   `json:"description"`
	Nodes       []string `json:"nodes"`
	Suggestion  string   `json:"suggestion"`
}

// PatternSummary provides aggregate statistics.
type PatternSummary struct {
	TotalPatterns    int            `json:"total_patterns"`
	ByType           map[string]int `json:"by_type"`
	AntiPatternCount int            `json:"anti_pattern_count"`
	HealthScore      float64        `json:"health_score"` // 0-100
}

// Detect runs all pattern detection algorithms.
func (d *Detector) Detect() *PatternReport {
	report := &PatternReport{
		Architectural: []ArchPattern{},
		Structural:    []StructPattern{},
		AntiPatterns:  []AntiPattern{},
	}

	// Detect architectural patterns
	report.Architectural = append(report.Architectural, d.detectFactoryPattern()...)
	report.Architectural = append(report.Architectural, d.detectSingletonPattern()...)
	report.Architectural = append(report.Architectural, d.detectHandlerPattern()...)
	report.Architectural = append(report.Architectural, d.detectRepositoryPattern()...)
	report.Architectural = append(report.Architectural, d.detectBuilderPattern()...)

	// Detect structural patterns
	report.Structural = append(report.Structural, d.detectHubNodes()...)
	report.Structural = append(report.Structural, d.detectLayeredArch()...)
	report.Structural = append(report.Structural, d.detectClusters()...)

	// Detect anti-patterns
	report.AntiPatterns = append(report.AntiPatterns, d.detectGodObjects()...)
	report.AntiPatterns = append(report.AntiPatterns, d.detectCircularDeps()...)
	report.AntiPatterns = append(report.AntiPatterns, d.detectDeadCode()...)

	// Calculate summary
	report.Summary = d.calculateSummary(report)

	return report
}

// detectFactoryPattern looks for factory functions (New* that return interfaces).
func (d *Detector) detectFactoryPattern() []ArchPattern {
	var patterns []ArchPattern

	for _, n := range d.nodes {
		if n.Type != "function" {
			continue
		}

		// Look for New* pattern
		if !strings.HasPrefix(n.Label, "New") {
			continue
		}

		// Check if it returns something
		sig := ""
		if n.Attrs != nil {
			sig = n.Attrs["signature"]
		}

		if sig == "" || !strings.Contains(sig, ")") {
			continue
		}

		// Find what it creates (outgoing edges)
		creates := []string{}
		for _, e := range d.outEdges[n.ID] {
			if e.Type == "returns" || e.Type == "creates" {
				creates = append(creates, e.To)
			}
		}

		pkg := ""
		if n.Attrs != nil {
			pkg = n.Attrs["package"]
		}

		patterns = append(patterns, ArchPattern{
			Type:        "factory",
			Name:        n.Label,
			Description: "Factory function that creates instances",
			Nodes:       append([]string{n.ID}, creates...),
			Confidence:  0.8,
			Location:    pkg,
		})
	}

	// Limit results
	if len(patterns) > 15 {
		patterns = patterns[:15]
	}

	return patterns
}

// detectSingletonPattern looks for singleton-like patterns.
func (d *Detector) detectSingletonPattern() []ArchPattern {
	var patterns []ArchPattern

	// Look for package-level variables that are widely used
	for _, n := range d.nodes {
		if n.Type != "variable" && n.Type != "constant" {
			continue
		}

		// Check if it's a global/package-level var
		label := strings.ToLower(n.Label)
		if !strings.Contains(label, "instance") &&
			!strings.Contains(label, "default") &&
			!strings.Contains(label, "global") {
			continue
		}

		// Check usage count
		usageCount := len(d.inEdges[n.ID])
		if usageCount < 3 {
			continue
		}

		pkg := ""
		if n.Attrs != nil {
			pkg = n.Attrs["package"]
		}

		patterns = append(patterns, ArchPattern{
			Type:        "singleton",
			Name:        n.Label,
			Description: "Global instance used across multiple locations",
			Nodes:       []string{n.ID},
			Confidence:  0.7,
			Location:    pkg,
		})
	}

	return patterns
}

// detectHandlerPattern looks for HTTP/RPC handler patterns.
func (d *Detector) detectHandlerPattern() []ArchPattern {
	var patterns []ArchPattern

	handlerKeywords := []string{"handle", "handler", "serve", "endpoint", "route"}

	for _, n := range d.nodes {
		if n.Type != "function" && n.Type != "method" {
			continue
		}

		label := strings.ToLower(n.Label)
		isHandler := false
		for _, kw := range handlerKeywords {
			if strings.Contains(label, kw) {
				isHandler = true
				break
			}
		}

		if !isHandler {
			continue
		}

		// Check signature for http.ResponseWriter or similar
		sig := ""
		if n.Attrs != nil {
			sig = n.Attrs["signature"]
		}

		confidence := 0.6
		if strings.Contains(sig, "ResponseWriter") || strings.Contains(sig, "Request") {
			confidence = 0.95
		} else if strings.Contains(sig, "Context") {
			confidence = 0.8
		}

		pkg := ""
		if n.Attrs != nil {
			pkg = n.Attrs["package"]
		}

		patterns = append(patterns, ArchPattern{
			Type:        "handler",
			Name:        n.Label,
			Description: "HTTP/RPC request handler",
			Nodes:       []string{n.ID},
			Confidence:  confidence,
			Location:    pkg,
		})
	}

	// Limit results
	if len(patterns) > 20 {
		patterns = patterns[:20]
	}

	return patterns
}

// detectRepositoryPattern looks for data access layer patterns.
func (d *Detector) detectRepositoryPattern() []ArchPattern {
	var patterns []ArchPattern

	repoKeywords := []string{"repo", "repository", "store", "dao", "storage"}

	for _, n := range d.nodes {
		if n.Type != "struct" && n.Type != "interface" {
			continue
		}

		label := strings.ToLower(n.Label)
		isRepo := false
		for _, kw := range repoKeywords {
			if strings.Contains(label, kw) {
				isRepo = true
				break
			}
		}

		if !isRepo {
			continue
		}

		// Find methods
		methods := []string{n.ID}
		for _, e := range d.outEdges[n.ID] {
			if e.Type == "has_method" || e.Type == "contains" {
				methods = append(methods, e.To)
			}
		}

		pkg := ""
		if n.Attrs != nil {
			pkg = n.Attrs["package"]
		}

		patterns = append(patterns, ArchPattern{
			Type:        "repository",
			Name:        n.Label,
			Description: "Data access layer component",
			Nodes:       methods,
			Confidence:  0.85,
			Location:    pkg,
		})
	}

	return patterns
}

// detectBuilderPattern looks for builder patterns.
func (d *Detector) detectBuilderPattern() []ArchPattern {
	var patterns []ArchPattern

	for _, n := range d.nodes {
		if n.Type != "struct" {
			continue
		}

		label := strings.ToLower(n.Label)
		if !strings.Contains(label, "builder") && !strings.Contains(label, "options") {
			continue
		}

		// Find methods that return the same type (fluent interface)
		fluentMethods := []string{}
		for _, e := range d.outEdges[n.ID] {
			if e.Type == "has_method" {
				method := d.nodeMap[e.To]
				if method != nil && method.Attrs != nil {
					sig := method.Attrs["signature"]
					if strings.Contains(sig, n.Label) {
						fluentMethods = append(fluentMethods, e.To)
					}
				}
			}
		}

		if len(fluentMethods) < 2 {
			continue
		}

		pkg := ""
		if n.Attrs != nil {
			pkg = n.Attrs["package"]
		}

		patterns = append(patterns, ArchPattern{
			Type:        "builder",
			Name:        n.Label,
			Description: "Builder pattern with fluent interface",
			Nodes:       append([]string{n.ID}, fluentMethods...),
			Confidence:  0.9,
			Location:    pkg,
		})
	}

	return patterns
}

// detectHubNodes finds highly connected nodes.
func (d *Detector) detectHubNodes() []StructPattern {
	var patterns []StructPattern

	type nodeScore struct {
		id    string
		in    int
		out   int
		total int
	}

	var scores []nodeScore
	for _, n := range d.nodes {
		in := len(d.inEdges[n.ID])
		out := len(d.outEdges[n.ID])
		total := in + out

		if total >= 10 {
			scores = append(scores, nodeScore{n.ID, in, out, total})
		}
	}

	// Sort by total connections
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].total > scores[j].total
	})

	// Take top 10
	if len(scores) > 10 {
		scores = scores[:10]
	}

	for _, s := range scores {
		patterns = append(patterns, StructPattern{
			Type:        "hub_node",
			Description: "Highly connected node (potential central component)",
			Nodes:       []string{s.id},
			Edges:       s.total,
		})
	}

	return patterns
}

// detectLayeredArch looks for layered architecture patterns.
func (d *Detector) detectLayeredArch() []StructPattern {
	var patterns []StructPattern

	// Group packages by layer keywords
	layers := map[string][]string{
		"presentation": {},
		"business":     {},
		"data":         {},
	}

	presentationKw := []string{"handler", "controller", "api", "http", "grpc", "rest"}
	businessKw := []string{"service", "usecase", "domain", "core", "business"}
	dataKw := []string{"repo", "repository", "store", "dao", "database", "db"}

	pkgNodes := make(map[string][]string)
	for _, n := range d.nodes {
		if n.Attrs == nil {
			continue
		}
		pkg := n.Attrs["package"]
		if pkg != "" {
			pkgNodes[pkg] = append(pkgNodes[pkg], n.ID)
		}
	}

	for pkg, nodes := range pkgNodes {
		pkgLower := strings.ToLower(pkg)

		for _, kw := range presentationKw {
			if strings.Contains(pkgLower, kw) {
				layers["presentation"] = append(layers["presentation"], nodes...)
				break
			}
		}
		for _, kw := range businessKw {
			if strings.Contains(pkgLower, kw) {
				layers["business"] = append(layers["business"], nodes...)
				break
			}
		}
		for _, kw := range dataKw {
			if strings.Contains(pkgLower, kw) {
				layers["data"] = append(layers["data"], nodes...)
				break
			}
		}
	}

	// Report if we found multiple layers
	foundLayers := 0
	for _, nodes := range layers {
		if len(nodes) > 0 {
			foundLayers++
		}
	}

	if foundLayers >= 2 {
		for layer, nodes := range layers {
			if len(nodes) > 0 {
				// Limit nodes per layer
				if len(nodes) > 10 {
					nodes = nodes[:10]
				}
				patterns = append(patterns, StructPattern{
					Type:        "layer_" + layer,
					Description: strings.Title(layer) + " layer components",
					Nodes:       nodes,
					Edges:       0,
				})
			}
		}
	}

	return patterns
}

// detectClusters finds tightly coupled node groups.
func (d *Detector) detectClusters() []StructPattern {
	// Simple clustering: nodes in same package with many internal connections
	var patterns []StructPattern

	// Group by package
	pkgNodes := make(map[string][]string)
	for _, n := range d.nodes {
		if n.Attrs == nil {
			continue
		}
		pkg := n.Attrs["package"]
		if pkg != "" {
			pkgNodes[pkg] = append(pkgNodes[pkg], n.ID)
		}
	}

	// Count internal edges per package
	for pkg, nodes := range pkgNodes {
		if len(nodes) < 5 {
			continue
		}

		nodeSet := make(map[string]bool)
		for _, n := range nodes {
			nodeSet[n] = true
		}

		internalEdges := 0
		for _, n := range nodes {
			for _, e := range d.outEdges[n] {
				if nodeSet[e.To] {
					internalEdges++
				}
			}
		}

		// High cohesion = many internal edges relative to node count
		cohesion := float64(internalEdges) / float64(len(nodes))
		if cohesion >= 2.0 {
			displayNodes := nodes
			if len(displayNodes) > 10 {
				displayNodes = displayNodes[:10]
			}

			patterns = append(patterns, StructPattern{
				Type:        "cluster",
				Description: "Tightly coupled cluster in " + pkg,
				Nodes:       displayNodes,
				Edges:       internalEdges,
			})
		}
	}

	return patterns
}

// detectGodObjects finds classes/structs with too many responsibilities.
func (d *Detector) detectGodObjects() []AntiPattern {
	var antiPatterns []AntiPattern

	for _, n := range d.nodes {
		if n.Type != "struct" && n.Type != "class" {
			continue
		}

		// Count methods
		methodCount := 0
		for _, e := range d.outEdges[n.ID] {
			if e.Type == "has_method" || e.Type == "contains" {
				targetNode := d.nodeMap[e.To]
				if targetNode != nil && (targetNode.Type == "method" || targetNode.Type == "function") {
					methodCount++
				}
			}
		}

		// Count dependencies
		depCount := len(d.outEdges[n.ID])

		if methodCount >= 20 || depCount >= 30 {
			severity := "medium"
			if methodCount >= 30 || depCount >= 50 {
				severity = "high"
			}

			antiPatterns = append(antiPatterns, AntiPattern{
				Type:        "god_object",
				Severity:    severity,
				Description: n.Label + " has too many responsibilities",
				Nodes:       []string{n.ID},
				Suggestion:  "Consider splitting into smaller, focused components",
			})
		}
	}

	return antiPatterns
}

// detectCircularDeps finds circular dependencies.
func (d *Detector) detectCircularDeps() []AntiPattern {
	var antiPatterns []AntiPattern

	// Simple cycle detection: A->B and B->A
	checked := make(map[string]bool)

	for _, e := range d.edges {
		if e.Type != "calls" && e.Type != "imports" && e.Type != "uses" {
			continue
		}

		key := e.From + "|" + e.To
		reverseKey := e.To + "|" + e.From

		if checked[key] {
			continue
		}
		checked[key] = true

		// Check for reverse edge
		for _, re := range d.outEdges[e.To] {
			if re.To == e.From && (re.Type == "calls" || re.Type == "imports" || re.Type == "uses") {
				if !checked[reverseKey] {
					checked[reverseKey] = true
					antiPatterns = append(antiPatterns, AntiPattern{
						Type:        "circular_dependency",
						Severity:    "high",
						Description: "Circular dependency between nodes",
						Nodes:       []string{e.From, e.To},
						Suggestion:  "Extract shared logic to break the cycle",
					})
				}
				break
			}
		}
	}

	// Limit results
	if len(antiPatterns) > 10 {
		antiPatterns = antiPatterns[:10]
	}

	return antiPatterns
}

// detectDeadCode finds potentially unused code.
func (d *Detector) detectDeadCode() []AntiPattern {
	var antiPatterns []AntiPattern

	for _, n := range d.nodes {
		if n.Type != "function" && n.Type != "method" {
			continue
		}

		// Skip common entry points
		label := strings.ToLower(n.Label)
		if label == "main" || label == "init" || strings.HasPrefix(label, "test") {
			continue
		}

		// Check if node has no incoming edges
		if len(d.inEdges[n.ID]) == 0 {
			// Check for exported (public) functions - these might be API
			if len(n.Label) > 0 && n.Label[0] >= 'A' && n.Label[0] <= 'Z' {
				continue // Skip exported functions
			}

			antiPatterns = append(antiPatterns, AntiPattern{
				Type:        "dead_code",
				Severity:    "low",
				Description: n.Label + " appears to be unused",
				Nodes:       []string{n.ID},
				Suggestion:  "Verify if this code is needed or can be removed",
			})
		}
	}

	// Limit results
	if len(antiPatterns) > 20 {
		antiPatterns = antiPatterns[:20]
	}

	return antiPatterns
}

// calculateSummary computes aggregate statistics.
func (d *Detector) calculateSummary(report *PatternReport) PatternSummary {
	byType := make(map[string]int)

	for _, p := range report.Architectural {
		byType[p.Type]++
	}
	for _, p := range report.Structural {
		byType[p.Type]++
	}

	totalPatterns := len(report.Architectural) + len(report.Structural)

	// Calculate health score (fewer anti-patterns = higher score)
	healthScore := 100.0
	for _, ap := range report.AntiPatterns {
		switch ap.Severity {
		case "high":
			healthScore -= 10
		case "medium":
			healthScore -= 5
		case "low":
			healthScore -= 2
		}
	}
	if healthScore < 0 {
		healthScore = 0
	}

	return PatternSummary{
		TotalPatterns:    totalPatterns,
		ByType:           byType,
		AntiPatternCount: len(report.AntiPatterns),
		HealthScore:      healthScore,
	}
}
