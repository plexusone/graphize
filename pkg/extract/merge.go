package extract

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
)

// SemanticExtraction represents the output from an LLM semantic extraction subagent.
type SemanticExtraction struct {
	Nodes []SemanticNode `json:"nodes"`
	Edges []SemanticEdge `json:"edges"`
}

// SemanticNode represents a node discovered by LLM analysis.
type SemanticNode struct {
	ID    string            `json:"id"`
	Type  string            `json:"type"`
	Label string            `json:"label"`
	Attrs map[string]string `json:"attrs,omitempty"`
}

// SemanticEdge represents an edge discovered by LLM analysis.
type SemanticEdge struct {
	From            string  `json:"from"`
	To              string  `json:"to"`
	Type            string  `json:"type"`
	Confidence      string  `json:"confidence"`
	ConfidenceScore float64 `json:"confidence_score"`
	Reason          string  `json:"reason"`
}

// ParseSemanticJSON parses JSON output from an LLM subagent.
func ParseSemanticJSON(data []byte) (*SemanticExtraction, error) {
	// Try to extract JSON from markdown code blocks if present
	jsonStr := string(data)
	if idx := strings.Index(jsonStr, "```json"); idx != -1 {
		start := idx + 7
		end := strings.Index(jsonStr[start:], "```")
		if end != -1 {
			jsonStr = jsonStr[start : start+end]
		}
	} else if idx := strings.Index(jsonStr, "```"); idx != -1 {
		start := idx + 3
		// Skip language identifier if present
		if newline := strings.Index(jsonStr[start:], "\n"); newline != -1 {
			start = start + newline + 1
		}
		end := strings.Index(jsonStr[start:], "```")
		if end != -1 {
			jsonStr = jsonStr[start : start+end]
		}
	}

	jsonStr = strings.TrimSpace(jsonStr)

	var result SemanticExtraction
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parsing semantic JSON: %w", err)
	}

	return &result, nil
}

// ValidateSemanticExtraction checks that the extraction is well-formed.
func ValidateSemanticExtraction(ext *SemanticExtraction) error {
	for i, e := range ext.Edges {
		if e.From == "" {
			return fmt.Errorf("edge %d: missing 'from' field", i)
		}
		if e.To == "" {
			return fmt.Errorf("edge %d: missing 'to' field", i)
		}
		if e.Type == "" {
			return fmt.Errorf("edge %d: missing 'type' field", i)
		}
		// Validate confidence
		switch e.Confidence {
		case "INFERRED", "AMBIGUOUS":
			// OK
		case "EXTRACTED":
			return fmt.Errorf("edge %d: LLM edges should not have EXTRACTED confidence", i)
		default:
			return fmt.Errorf("edge %d: invalid confidence '%s'", i, e.Confidence)
		}
		// Validate score
		if e.ConfidenceScore < 0 || e.ConfidenceScore > 1 {
			return fmt.Errorf("edge %d: confidence_score must be 0.0-1.0, got %f", i, e.ConfidenceScore)
		}
	}
	return nil
}

// MergeExtractions combines AST extraction with semantic extraction.
// AST edges take precedence (they have EXTRACTED confidence).
// Semantic edges are added if they don't duplicate AST edges.
func MergeExtractions(astNodes []*graph.Node, astEdges []*graph.Edge, semantic *SemanticExtraction) ([]*graph.Node, []*graph.Edge) {
	// Build set of existing edges for deduplication
	existingEdges := make(map[string]bool)
	for _, e := range astEdges {
		key := edgeKey(e.From, e.To, e.Type)
		existingEdges[key] = true
	}

	// Build set of existing nodes
	existingNodes := make(map[string]bool)
	for _, n := range astNodes {
		existingNodes[n.ID] = true
	}

	// Merge nodes (add new ones only)
	mergedNodes := make([]*graph.Node, len(astNodes))
	copy(mergedNodes, astNodes)

	for _, sn := range semantic.Nodes {
		if !existingNodes[sn.ID] {
			mergedNodes = append(mergedNodes, &graph.Node{
				ID:    sn.ID,
				Type:  sn.Type,
				Label: sn.Label,
				Attrs: sn.Attrs,
			})
			existingNodes[sn.ID] = true
		}
	}

	// Merge edges (add non-duplicate semantic edges)
	mergedEdges := make([]*graph.Edge, len(astEdges))
	copy(mergedEdges, astEdges)

	for _, se := range semantic.Edges {
		key := edgeKey(se.From, se.To, se.Type)
		if !existingEdges[key] {
			conf := graph.ConfidenceInferred
			if se.Confidence == "AMBIGUOUS" {
				conf = graph.ConfidenceAmbiguous
			}

			mergedEdges = append(mergedEdges, &graph.Edge{
				From:            se.From,
				To:              se.To,
				Type:            se.Type,
				Confidence:      conf,
				ConfidenceScore: se.ConfidenceScore,
				Attrs: map[string]string{
					"reason": se.Reason,
					"source": "llm",
				},
			})
			existingEdges[key] = true
		}
	}

	return mergedNodes, mergedEdges
}

// edgeKey creates a unique key for edge deduplication.
func edgeKey(from, to, edgeType string) string {
	return from + "|" + to + "|" + edgeType
}

// SemanticEdgeTypes are the valid edge types for LLM-inferred relationships.
var SemanticEdgeTypes = []string{
	"inferred_depends",   // Implicit dependency
	"rationale_for",      // Design rationale from comments
	"similar_to",         // Semantic similarity
	"implements_pattern", // Design pattern implementation
	"shared_concern",     // Cross-cutting concern
}

// IsValidSemanticEdgeType checks if an edge type is valid for semantic edges.
func IsValidSemanticEdgeType(edgeType string) bool {
	for _, t := range SemanticEdgeTypes {
		if t == edgeType {
			return true
		}
	}
	return false
}

// ChunkFiles splits a list of files into chunks of the specified size.
func ChunkFiles(files []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(files); i += chunkSize {
		end := i + chunkSize
		if end > len(files) {
			end = len(files)
		}
		chunks = append(chunks, files[i:end])
	}
	return chunks
}

// BuildSubagentPrompt creates the prompt for a semantic extraction subagent.
func BuildSubagentPrompt(files []string, chunkID, totalChunks int, baseDir string) string {
	var sb strings.Builder

	sb.WriteString("You are a graphize semantic extraction subagent. Your task is to analyze Go source files\n")
	sb.WriteString("and discover relationships that are NOT visible in the Abstract Syntax Tree (AST).\n\n")

	sb.WriteString(fmt.Sprintf("## Context\n\nYou are processing chunk %d of %d.\n", chunkID, totalChunks))
	sb.WriteString(fmt.Sprintf("Base directory: %s\n\n", baseDir))

	sb.WriteString("## Files to Analyze\n\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("- %s\n", f))
	}

	sb.WriteString("\n## What to Extract\n\n")
	sb.WriteString("The AST already captures imports, function calls, type definitions. You should discover:\n\n")
	sb.WriteString("1. **inferred_depends**: Implicit dependencies not explicit in imports\n")
	sb.WriteString("2. **rationale_for**: Design decisions explained in comments\n")
	sb.WriteString("3. **similar_to**: Functions/types solving similar problems\n")
	sb.WriteString("4. **implements_pattern**: Design pattern implementations\n")
	sb.WriteString("5. **shared_concern**: Cross-cutting concerns (logging, auth, etc.)\n\n")

	sb.WriteString("## Confidence Scoring\n\n")
	sb.WriteString("- 0.8-1.0: Very confident (clear evidence)\n")
	sb.WriteString("- 0.6-0.8: Moderately confident\n")
	sb.WriteString("- 0.3-0.6: Low confidence\n")
	sb.WriteString("- Below 0.3: Use AMBIGUOUS confidence\n\n")

	sb.WriteString("## Output Format\n\n")
	sb.WriteString("Return ONLY valid JSON (no markdown, no explanation):\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{
  "nodes": [],
  "edges": [
    {
      "from": "func_filename.go.FunctionName",
      "to": "type_TypeName",
      "type": "inferred_depends",
      "confidence": "INFERRED",
      "confidence_score": 0.75,
      "reason": "Both handle user authentication flow"
    }
  ]
}
`)
	sb.WriteString("```\n\n")

	sb.WriteString("## Important\n\n")
	sb.WriteString("- Only output relationships you are confident about\n")
	sb.WriteString("- Always include a clear reason\n")
	sb.WriteString("- Do NOT duplicate AST relationships (imports, calls, contains)\n")
	sb.WriteString("- Node IDs must match graphize convention: type_name (e.g., func_main.go.HandleRequest)\n")
	sb.WriteString("- Return empty edges array if no semantic relationships found\n")

	return sb.String()
}
