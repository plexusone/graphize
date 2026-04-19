package analyze

import (
	"fmt"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/pkg/metrics"
)

// CorpusHealth represents the health assessment of a knowledge graph corpus.
type CorpusHealth struct {
	// FileCount is the number of source files analyzed.
	FileCount int `json:"file_count"`

	// WordCount is the estimated word count in source files.
	WordCount int `json:"word_count"`

	// EstimatedTokens is the estimated token count for source files.
	EstimatedTokens int `json:"estimated_tokens"`

	// GraphNodes is the number of nodes in the graph.
	GraphNodes int `json:"graph_nodes"`

	// GraphEdges is the number of edges in the graph.
	GraphEdges int `json:"graph_edges"`

	// GraphTokens is the estimated token count for graph representation.
	GraphTokens int `json:"graph_tokens"`

	// TokenReduction is the percentage reduction in tokens (0-100).
	// A value of 75 means the graph uses 75% fewer tokens than raw source.
	TokenReduction float64 `json:"token_reduction"`

	// Verdict is the overall health assessment: "valuable", "marginal", or "limited".
	Verdict string `json:"verdict"`

	// VerdictReason explains the verdict in human-readable terms.
	VerdictReason string `json:"verdict_reason"`
}

// HealthOptions configures corpus health assessment.
type HealthOptions struct {
	// IncludeFileContent indicates whether to count tokens from source files.
	// If false, uses estimates based on node/edge counts.
	IncludeFileContent bool

	// SourceTokens is the pre-computed source token count (if known).
	// If set, this overrides file walking.
	SourceTokens int

	// FileCount is the pre-computed file count (if known).
	FileCount int
}

// CheckCorpusHealth assesses the health and value of a knowledge graph.
// It compares the graph representation against the source code to determine
// if the graph provides meaningful token reduction and structural insight.
func CheckCorpusHealth(nodes []*graph.Node, edges []*graph.Edge, opts HealthOptions) *CorpusHealth {
	health := &CorpusHealth{
		GraphNodes: len(nodes),
		GraphEdges: len(edges),
		FileCount:  opts.FileCount,
	}

	// Estimate graph tokens (rough: ~20 tokens per node, ~10 per edge)
	health.GraphTokens = estimateGraphTokens(nodes, edges)

	// Use provided source tokens or estimate from file count
	if opts.SourceTokens > 0 {
		health.EstimatedTokens = opts.SourceTokens
	} else if opts.FileCount > 0 {
		// Rough estimate: ~500 tokens per source file on average
		health.EstimatedTokens = opts.FileCount * 500
	}

	// Estimate word count (tokens * 0.7 roughly)
	health.WordCount = int(float64(health.EstimatedTokens) * 0.7)

	// Calculate token reduction
	if health.EstimatedTokens > 0 {
		reduction := float64(health.EstimatedTokens-health.GraphTokens) / float64(health.EstimatedTokens) * 100
		if reduction < 0 {
			reduction = 0
		}
		health.TokenReduction = reduction
	}

	// Determine verdict
	health.Verdict, health.VerdictReason = determineVerdict(health)

	return health
}

// CheckCorpusHealthFromSource calculates health metrics by walking source files.
func CheckCorpusHealthFromSource(nodes []*graph.Node, edges []*graph.Edge, sourceDir string) (*CorpusHealth, error) {
	var totalTokens int
	var fileCount int

	opts := metrics.DefaultWalkOptions()
	err := metrics.WalkSourceFilesWithContent(sourceDir, opts, func(path string, content []byte) error {
		totalTokens += metrics.EstimateTokensInFile(content)
		fileCount++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking source files: %w", err)
	}

	healthOpts := HealthOptions{
		SourceTokens: totalTokens,
		FileCount:    fileCount,
	}

	return CheckCorpusHealth(nodes, edges, healthOpts), nil
}

// estimateGraphTokens estimates the token count for graph representation.
func estimateGraphTokens(nodes []*graph.Node, edges []*graph.Edge) int {
	tokens := 0

	// Each node: ID (~3-5 tokens), type (~1), label (~2-4), attrs (~5-10)
	// Rough average: ~15 tokens per node
	for _, n := range nodes {
		tokens += 5 // Base: id + type
		if n.Label != "" {
			tokens += estimateLabelTokens(n.Label)
		}
		tokens += len(n.Attrs) * 3 // ~3 tokens per attribute
	}

	// Each edge: from (~3), to (~3), type (~1), confidence (~1), attrs (~3)
	// Rough average: ~10 tokens per edge
	for _, e := range edges {
		tokens += 8 // Base: from + to + type + confidence
		tokens += len(e.Attrs) * 3
	}

	return tokens
}

// estimateLabelTokens estimates tokens for a label.
func estimateLabelTokens(label string) int {
	// CamelCase and snake_case split into multiple tokens
	words := 1
	for _, r := range label {
		if r == '_' || r == '-' || r == '.' || r == '/' {
			words++
		}
	}
	// Roughly 1.5 tokens per word
	return int(float64(words) * 1.5)
}

// determineVerdict assesses the corpus and returns verdict + reason.
func determineVerdict(h *CorpusHealth) (string, string) {
	// Check for minimal corpus
	if h.GraphNodes < 10 {
		return "limited", "Graph has too few nodes (<10) to provide meaningful structure."
	}

	if h.GraphEdges < 5 {
		return "limited", "Graph has too few edges (<5) to show relationships."
	}

	// Check edge-to-node ratio (healthy graphs have >0.5 ratio)
	ratio := float64(h.GraphEdges) / float64(h.GraphNodes)
	if ratio < 0.3 {
		return "marginal", fmt.Sprintf("Low edge density (%.2f edges/node). Graph may be fragmented.", ratio)
	}

	// Check token reduction
	if h.TokenReduction >= 70 {
		return "valuable", fmt.Sprintf("%.0f%% token reduction with %.1f edges/node - highly effective graph.", h.TokenReduction, ratio)
	}

	if h.TokenReduction >= 50 {
		return "valuable", fmt.Sprintf("%.0f%% token reduction - graph provides significant context compression.", h.TokenReduction)
	}

	if h.TokenReduction >= 30 {
		return "marginal", fmt.Sprintf("%.0f%% token reduction - graph provides moderate value.", h.TokenReduction)
	}

	if h.TokenReduction >= 10 {
		return "marginal", fmt.Sprintf("%.0f%% token reduction - consider enriching with semantic edges.", h.TokenReduction)
	}

	return "limited", fmt.Sprintf("Only %.0f%% token reduction - raw source may be more useful.", h.TokenReduction)
}

// FormatHealth formats health assessment as human-readable text.
func FormatHealth(h *CorpusHealth) string {
	var sb strings.Builder

	sb.WriteString("## Corpus Health Assessment\n\n")

	// Verdict banner
	var verdictIcon string
	switch h.Verdict {
	case "valuable":
		verdictIcon = "[+]"
	case "marginal":
		verdictIcon = "[~]"
	case "limited":
		verdictIcon = "[-]"
	}
	fmt.Fprintf(&sb, "**Verdict:** %s %s\n\n", verdictIcon, strings.ToUpper(h.Verdict))
	fmt.Fprintf(&sb, "*%s*\n\n", h.VerdictReason)

	// Metrics table
	sb.WriteString("### Metrics\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	fmt.Fprintf(&sb, "| Source Files | %d |\n", h.FileCount)
	fmt.Fprintf(&sb, "| Source Tokens | ~%d |\n", h.EstimatedTokens)
	fmt.Fprintf(&sb, "| Graph Nodes | %d |\n", h.GraphNodes)
	fmt.Fprintf(&sb, "| Graph Edges | %d |\n", h.GraphEdges)
	fmt.Fprintf(&sb, "| Graph Tokens | ~%d |\n", h.GraphTokens)
	fmt.Fprintf(&sb, "| Token Reduction | %.1f%% |\n", h.TokenReduction)

	if h.GraphNodes > 0 {
		ratio := float64(h.GraphEdges) / float64(h.GraphNodes)
		fmt.Fprintf(&sb, "| Edge Density | %.2f edges/node |\n", ratio)
	}

	sb.WriteString("\n")

	return sb.String()
}
