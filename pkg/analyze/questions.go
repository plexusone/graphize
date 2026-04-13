// Package analyze provides graph analysis functions.
package analyze

import (
	"fmt"
	"sort"

	"github.com/plexusone/graphfs/pkg/graph"
)

// Question represents a suggested question about the codebase.
type Question struct {
	Type     string `json:"type"`
	Question string `json:"question"`
	Why      string `json:"why"`
}

// SuggestQuestions generates questions based on graph analysis.
// Questions help identify areas needing human review or documentation.
func SuggestQuestions(nodes []*graph.Node, edges []*graph.Edge, communities map[int][]string, topN int) []Question {
	var questions []Question

	// Build adjacency and degree maps
	nodeMap := make(map[string]*graph.Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	inDegree := make(map[string]int)
	outDegree := make(map[string]int)
	for _, e := range edges {
		outDegree[e.From]++
		inDegree[e.To]++
	}

	// Build reverse community map (node -> community ID)
	nodeToCommunity := make(map[string]int)
	for cid, members := range communities {
		for _, m := range members {
			nodeToCommunity[m] = cid
		}
	}

	// 1. Ambiguous edges - need human review
	ambiguousEdges := findAmbiguousEdges(edges, nodeMap)
	if len(ambiguousEdges) > 0 {
		examples := ambiguousEdges
		if len(examples) > 3 {
			examples = examples[:3]
		}
		var exampleStrs []string
		for _, e := range examples {
			fromLabel := labelFor(nodeMap, e.From)
			toLabel := labelFor(nodeMap, e.To)
			exampleStrs = append(exampleStrs, fmt.Sprintf("`%s` → `%s`", fromLabel, toLabel))
		}
		questions = append(questions, Question{
			Type:     "ambiguous_edges",
			Question: fmt.Sprintf("Are the %d AMBIGUOUS relationships correct? (e.g., %s)", len(ambiguousEdges), joinMax(exampleStrs, 2)),
			Why:      fmt.Sprintf("%d edges have AMBIGUOUS confidence - model-uncertain connections that need human verification.", len(ambiguousEdges)),
		})
	}

	// 2. Bridge nodes - connect different communities
	bridges := findBridgeNodes(nodes, edges, nodeToCommunity, nodeMap)
	if len(bridges) > 0 {
		labels := make([]string, 0, 3)
		for i, b := range bridges {
			if i >= 3 {
				break
			}
			labels = append(labels, fmt.Sprintf("`%s`", labelFor(nodeMap, b.ID)))
		}
		questions = append(questions, Question{
			Type:     "bridge_nodes",
			Question: fmt.Sprintf("Why do %s connect multiple communities? Are they intentional integration points?", joinMax(labels, 3)),
			Why:      fmt.Sprintf("%d bridge nodes span community boundaries - potential architectural coupling or intentional integration points.", len(bridges)),
		})
	}

	// 3. Inferred edges - need verification
	inferredByNode := findInferredEdgesByNode(edges, nodeMap)
	for nodeID, inferredEdges := range inferredByNode {
		if len(inferredEdges) < 3 {
			continue
		}
		label := labelFor(nodeMap, nodeID)
		others := make([]string, 0, 2)
		for i, e := range inferredEdges {
			if i >= 2 {
				break
			}
			otherID := e.To
			if e.To == nodeID {
				otherID = e.From
			}
			others = append(others, fmt.Sprintf("`%s`", labelFor(nodeMap, otherID)))
		}
		questions = append(questions, Question{
			Type:     "verify_inferred",
			Question: fmt.Sprintf("Are the %d inferred relationships involving `%s` (e.g., with %s) actually correct?", len(inferredEdges), label, joinMax(others, 2)),
			Why:      fmt.Sprintf("`%s` has %d INFERRED edges - LLM-reasoned connections that may need verification.", label, len(inferredEdges)),
		})
		if len(questions) >= topN {
			break
		}
	}

	// 4. Isolated nodes - documentation gaps
	isolated := findIsolatedNodes(nodes, edges, nodeMap)
	if len(isolated) > 0 {
		labels := make([]string, 0, 3)
		for i, n := range isolated {
			if i >= 3 {
				break
			}
			labels = append(labels, fmt.Sprintf("`%s`", labelFor(nodeMap, n.ID)))
		}
		questions = append(questions, Question{
			Type:     "isolated_nodes",
			Question: fmt.Sprintf("What connects %s to the rest of the system?", joinMax(labels, 3)),
			Why:      fmt.Sprintf("%d weakly-connected nodes found - possible documentation gaps or missing edges.", len(isolated)),
		})
	}

	// 5. Low cohesion communities - may need splitting
	for cid, members := range communities {
		if len(members) < 5 {
			continue
		}
		// Calculate cohesion for this community
		adj := buildAdjacency(edges)
		score := CohesionScore(members, adj)
		if score < 0.15 {
			label := fmt.Sprintf("Community %d", cid)
			// Try to find a better label from the members
			for _, m := range members {
				if n, ok := nodeMap[m]; ok && n.Type == "package" {
					label = labelFor(nodeMap, m)
					break
				}
			}
			questions = append(questions, Question{
				Type:     "low_cohesion",
				Question: fmt.Sprintf("Should `%s` (%d nodes) be split into smaller, more focused modules?", label, len(members)),
				Why:      fmt.Sprintf("Cohesion score %.2f - nodes in this community are weakly interconnected.", score),
			})
		}
	}

	if len(questions) == 0 {
		return []Question{{
			Type:     "no_signal",
			Question: "",
			Why:      "Not enough signal to generate questions. The corpus has no AMBIGUOUS edges, no bridge nodes, no INFERRED relationships, and all communities are tightly cohesive.",
		}}
	}

	// Sort by type priority and limit
	sort.Slice(questions, func(i, j int) bool {
		return questionPriority(questions[i].Type) < questionPriority(questions[j].Type)
	})

	if len(questions) > topN {
		questions = questions[:topN]
	}

	return questions
}

func questionPriority(qtype string) int {
	switch qtype {
	case "ambiguous_edges":
		return 1
	case "bridge_nodes":
		return 2
	case "verify_inferred":
		return 3
	case "isolated_nodes":
		return 4
	case "low_cohesion":
		return 5
	default:
		return 10
	}
}

func findAmbiguousEdges(edges []*graph.Edge, _ map[string]*graph.Node) []*graph.Edge {
	var result []*graph.Edge
	for _, e := range edges {
		if e.Confidence == graph.ConfidenceAmbiguous {
			result = append(result, e)
		}
	}
	return result
}

type bridgeNode struct {
	ID          string
	Communities map[int]bool
}

func findBridgeNodes(_ []*graph.Node, edges []*graph.Edge, nodeToCommunity map[string]int, nodeMap map[string]*graph.Node) []bridgeNode {
	// Find nodes that connect to multiple communities
	nodeConnections := make(map[string]map[int]bool)

	for _, e := range edges {
		fromCom := nodeToCommunity[e.From]
		toCom := nodeToCommunity[e.To]

		if nodeConnections[e.From] == nil {
			nodeConnections[e.From] = make(map[int]bool)
		}
		nodeConnections[e.From][fromCom] = true
		nodeConnections[e.From][toCom] = true

		if nodeConnections[e.To] == nil {
			nodeConnections[e.To] = make(map[int]bool)
		}
		nodeConnections[e.To][fromCom] = true
		nodeConnections[e.To][toCom] = true
	}

	var bridges []bridgeNode
	for nodeID, coms := range nodeConnections {
		if len(coms) >= 2 {
			// Skip file nodes - they're expected to bridge
			if n, ok := nodeMap[nodeID]; ok && n.Type == "file" {
				continue
			}
			bridges = append(bridges, bridgeNode{ID: nodeID, Communities: coms})
		}
	}

	// Sort by number of communities bridged (descending)
	sort.Slice(bridges, func(i, j int) bool {
		return len(bridges[i].Communities) > len(bridges[j].Communities)
	})

	return bridges
}

func findInferredEdgesByNode(edges []*graph.Edge, _ map[string]*graph.Node) map[string][]*graph.Edge {
	result := make(map[string][]*graph.Edge)
	for _, e := range edges {
		if e.Confidence == graph.ConfidenceInferred {
			result[e.From] = append(result[e.From], e)
			result[e.To] = append(result[e.To], e)
		}
	}
	return result
}

func findIsolatedNodes(nodes []*graph.Node, edges []*graph.Edge, _ map[string]*graph.Node) []*graph.Node {
	degree := make(map[string]int)
	for _, e := range edges {
		degree[e.From]++
		degree[e.To]++
	}

	var isolated []*graph.Node
	for _, n := range nodes {
		// Skip file, package, and concept nodes - they're expected to have few connections
		if n.Type == "file" || n.Type == "package" || n.Type == "concept" {
			continue
		}
		if degree[n.ID] <= 1 {
			isolated = append(isolated, n)
		}
	}
	return isolated
}

func buildAdjacency(edges []*graph.Edge) map[string]map[string]bool {
	adj := make(map[string]map[string]bool)
	for _, e := range edges {
		if adj[e.From] == nil {
			adj[e.From] = make(map[string]bool)
		}
		if adj[e.To] == nil {
			adj[e.To] = make(map[string]bool)
		}
		adj[e.From][e.To] = true
		adj[e.To][e.From] = true
	}
	return adj
}

func labelFor(nodeMap map[string]*graph.Node, id string) string {
	if n, ok := nodeMap[id]; ok && n.Label != "" {
		return n.Label
	}
	return id
}

func joinMax(items []string, max int) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) <= max {
		result := items[0]
		for i := 1; i < len(items); i++ {
			if i == len(items)-1 {
				result += " and " + items[i]
			} else {
				result += ", " + items[i]
			}
		}
		return result
	}
	result := items[0]
	for i := 1; i < max; i++ {
		result += ", " + items[i]
	}
	return result
}
