package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphfs/pkg/store"
	"github.com/plexusone/graphize/pkg/analyze"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for graph queries",
	Long: `Start a Model Context Protocol (MCP) server that exposes graph query tools.

The server supports multiple projects - pass the project path with each query.
Graphs are loaded on-demand and cached for performance.

Tools available:
  - list_projects: Discover projects with .graphize directories
  - query_graph: Search and traverse the graph
  - get_node: Get details for a specific node
  - get_neighbors: Get all neighbors of a node
  - get_community: Get nodes in a community
  - graph_summary: Get overall graph statistics

The server runs over stdio and can be used with Claude Desktop, Claude Code,
or any MCP-compatible client.

Example Claude Desktop config:
  {
    "mcpServers": {
      "graphize": {
        "command": "graphize",
        "args": ["serve"]
      }
    }
  }`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

// MultiGraphServer manages multiple project graphs with lazy loading.
type MultiGraphServer struct {
	mu     sync.RWMutex
	graphs map[string]*GraphServer // path -> loaded graph
}

func newMultiGraphServer() *MultiGraphServer {
	return &MultiGraphServer{
		graphs: make(map[string]*GraphServer),
	}
}

// getGraph returns a cached graph or loads it on demand.
func (mgs *MultiGraphServer) getGraph(projectPath string) (*GraphServer, error) {
	// Normalize path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check for .graphize subdirectory
	graphPath := absPath
	if !strings.HasSuffix(absPath, ".graphize") {
		graphPath = filepath.Join(absPath, ".graphize")
	}

	// Check if graph directory exists
	if _, err := os.Stat(graphPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no .graphize directory found at %s (run 'graphize analyze' first)", projectPath)
	}

	// Check cache
	mgs.mu.RLock()
	if gs, ok := mgs.graphs[graphPath]; ok {
		mgs.mu.RUnlock()
		return gs, nil
	}
	mgs.mu.RUnlock()

	// Load graph
	gs, err := newGraphServer(graphPath)
	if err != nil {
		return nil, err
	}

	// Cache it
	mgs.mu.Lock()
	mgs.graphs[graphPath] = gs
	mgs.mu.Unlock()

	return gs, nil
}

// GraphServer holds the loaded graph data for a single project.
type GraphServer struct {
	path        string
	nodes       []*graph.Node
	edges       []*graph.Edge
	nodeMap     map[string]*graph.Node
	adj         map[string][]string // adjacency list
	communities map[int][]string
}

func newGraphServer(graphPath string) (*GraphServer, error) {
	s, err := store.NewFSStore(graphPath)
	if err != nil {
		return nil, fmt.Errorf("opening graph store: %w", err)
	}

	nodes, err := s.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	edges, err := s.ListEdges()
	if err != nil {
		return nil, fmt.Errorf("listing edges: %w", err)
	}

	// Build lookup structures
	nodeMap := make(map[string]*graph.Node)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	adj := make(map[string][]string)
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e.To)
		adj[e.To] = append(adj[e.To], e.From)
	}

	// Detect communities
	clusterResult := analyze.DetectCommunities(nodes, edges)
	communities := make(map[int][]string)
	for _, c := range clusterResult.Communities {
		communities[c.ID] = c.Members
	}

	return &GraphServer{
		path:        graphPath,
		nodes:       nodes,
		edges:       edges,
		nodeMap:     nodeMap,
		adj:         adj,
		communities: communities,
	}, nil
}

// Tool input/output types

type ListProjectsInput struct {
	SearchPath string `json:"search_path" jsonschema:"description=Directory to search for projects with .graphize directories"`
	MaxDepth   int    `json:"max_depth,omitempty" jsonschema:"description=Maximum directory depth to search (default 3)"`
}

type ListProjectsOutput struct {
	Projects []ProjectInfo `json:"projects"`
	Total    int           `json:"total"`
}

type ProjectInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Nodes int    `json:"nodes"`
	Edges int    `json:"edges"`
}

type QueryGraphInput struct {
	Path       string `json:"path" jsonschema:"description=Project directory path (will look for .graphize subdirectory)"`
	Query      string `json:"query" jsonschema:"description=Search terms or node label to find"`
	Mode       string `json:"mode,omitempty" jsonschema:"description=Traversal mode: bfs (broad) or dfs (deep),enum=bfs;dfs,default=bfs"`
	Depth      int    `json:"depth,omitempty" jsonschema:"description=Traversal depth (1-6),default=2"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum results to return,default=20"`
}

type QueryGraphOutput struct {
	Project string     `json:"project"`
	Nodes   []NodeInfo `json:"nodes"`
	Edges   []EdgeInfo `json:"edges"`
	Summary string     `json:"summary"`
}

type GetNodeInput struct {
	Path string `json:"path" jsonschema:"description=Project directory path"`
	ID   string `json:"id" jsonschema:"description=Node ID or label to look up"`
}

type GetNodeOutput struct {
	Project string      `json:"project"`
	Node    *NodeDetail `json:"node,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type GetNeighborsInput struct {
	Path      string `json:"path" jsonschema:"description=Project directory path"`
	ID        string `json:"id" jsonschema:"description=Node ID to get neighbors for"`
	Direction string `json:"direction,omitempty" jsonschema:"description=Edge direction: in out or both,enum=in;out;both,default=both"`
}

type GetNeighborsOutput struct {
	Project   string         `json:"project"`
	Neighbors []NeighborInfo `json:"neighbors"`
	Total     int            `json:"total"`
}

type GetCommunityInput struct {
	Path string `json:"path" jsonschema:"description=Project directory path"`
	ID   int    `json:"id" jsonschema:"description=Community ID"`
}

type GetCommunityOutput struct {
	Project string     `json:"project"`
	Members []NodeInfo `json:"members"`
	Size    int        `json:"size"`
	Label   string     `json:"label"`
}

type GraphSummaryInput struct {
	Path string `json:"path" jsonschema:"description=Project directory path"`
}

type GraphSummaryOutput struct {
	Project            string         `json:"project"`
	TotalNodes         int            `json:"total_nodes"`
	TotalEdges         int            `json:"total_edges"`
	NodeTypes          map[string]int `json:"node_types"`
	EdgeTypes          map[string]int `json:"edge_types"`
	Communities        int            `json:"communities"`
	GodNodes           []NodeInfo     `json:"god_nodes"`
	EdgeConfidence     map[string]int `json:"edge_confidence"`
	SuggestedQuestions []string       `json:"suggested_questions,omitempty"`
}

// Info types for output
type NodeInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type NodeDetail struct {
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	Type      string            `json:"type"`
	Attrs     map[string]string `json:"attrs,omitempty"`
	InDegree  int               `json:"in_degree"`
	OutDegree int               `json:"out_degree"`
	Community int               `json:"community,omitempty"`
}

type EdgeInfo struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	Confidence string `json:"confidence,omitempty"`
}

type NeighborInfo struct {
	Node      NodeInfo `json:"node"`
	EdgeType  string   `json:"edge_type"`
	Direction string   `json:"direction"`
}

func runServe(cmd *cobra.Command, args []string) error {
	mgs := newMultiGraphServer()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "graphize",
		Version: "0.2.0",
	}, nil)

	// Register tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_projects",
		Description: "Discover projects with .graphize directories in a search path. Use this to find available knowledge graphs.",
	}, mgs.listProjects)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_graph",
		Description: "Search the knowledge graph and traverse from matching nodes. Returns relevant nodes and edges.",
	}, mgs.queryGraph)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_node",
		Description: "Get full details for a specific node by ID or label.",
	}, mgs.getNode)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_neighbors",
		Description: "Get all direct neighbors of a node with edge details.",
	}, mgs.getNeighbors)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_community",
		Description: "Get all nodes in a specific community.",
	}, mgs.getCommunity)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "graph_summary",
		Description: "Get overall graph statistics including node/edge counts, types, communities, and suggested questions.",
	}, mgs.graphSummary)

	// Run server over stdio
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

func (mgs *MultiGraphServer) listProjects(ctx context.Context, req *mcp.CallToolRequest, input ListProjectsInput) (*mcp.CallToolResult, ListProjectsOutput, error) {
	if input.MaxDepth == 0 {
		input.MaxDepth = 3
	}

	searchPath, err := filepath.Abs(input.SearchPath)
	if err != nil {
		return nil, ListProjectsOutput{}, fmt.Errorf("invalid search path: %w", err)
	}

	var projects []ProjectInfo

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible directories
		}

		// Check depth
		rel, _ := filepath.Rel(searchPath, path)
		depth := strings.Count(rel, string(filepath.Separator))
		if depth > input.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Look for .graphize directories
		if info.IsDir() && info.Name() == ".graphize" {
			projectPath := filepath.Dir(path)
			projectName := filepath.Base(projectPath)

			// Try to get node/edge counts
			nodes, edges := 0, 0
			if gs, err := mgs.getGraph(projectPath); err == nil {
				nodes = len(gs.nodes)
				edges = len(gs.edges)
			}

			projects = append(projects, ProjectInfo{
				Name:  projectName,
				Path:  projectPath,
				Nodes: nodes,
				Edges: edges,
			})

			return filepath.SkipDir // Don't descend into .graphize
		}

		return nil
	})

	if err != nil {
		return nil, ListProjectsOutput{}, fmt.Errorf("scanning directories: %w", err)
	}

	return nil, ListProjectsOutput{
		Projects: projects,
		Total:    len(projects),
	}, nil
}

func (mgs *MultiGraphServer) queryGraph(ctx context.Context, req *mcp.CallToolRequest, input QueryGraphInput) (*mcp.CallToolResult, QueryGraphOutput, error) {
	gs, err := mgs.getGraph(input.Path)
	if err != nil {
		return nil, QueryGraphOutput{
			Project: input.Path,
			Summary: fmt.Sprintf("Error: %v", err),
		}, nil
	}

	// Set defaults
	if input.Mode == "" {
		input.Mode = "bfs"
	}
	if input.Depth == 0 {
		input.Depth = 2
	}
	if input.Depth > 6 {
		input.Depth = 6
	}
	if input.MaxResults == 0 {
		input.MaxResults = 20
	}

	// Find starting nodes matching query
	query := strings.ToLower(input.Query)
	var startNodes []string
	for _, n := range gs.nodes {
		label := strings.ToLower(n.Label)
		id := strings.ToLower(n.ID)
		if strings.Contains(label, query) || strings.Contains(id, query) {
			startNodes = append(startNodes, n.ID)
		}
	}

	if len(startNodes) == 0 {
		return nil, QueryGraphOutput{
			Project: filepath.Base(input.Path),
			Summary: fmt.Sprintf("No nodes found matching '%s'", input.Query),
		}, nil
	}

	// Limit starting nodes
	if len(startNodes) > 5 {
		startNodes = startNodes[:5]
	}

	// Traverse
	visited := make(map[string]bool)
	var edgesFound []*graph.Edge

	if input.Mode == "dfs" {
		gs.dfs(startNodes, input.Depth, visited, &edgesFound)
	} else {
		gs.bfs(startNodes, input.Depth, visited, &edgesFound)
	}

	// Build output
	var nodes []NodeInfo
	for id := range visited {
		if n, ok := gs.nodeMap[id]; ok {
			nodes = append(nodes, NodeInfo{
				ID:    n.ID,
				Label: n.Label,
				Type:  n.Type,
			})
		}
	}

	// Limit results
	if len(nodes) > input.MaxResults {
		nodes = nodes[:input.MaxResults]
	}

	var edges []EdgeInfo
	for _, e := range edgesFound {
		if visited[e.From] && visited[e.To] {
			edges = append(edges, EdgeInfo{
				From:       e.From,
				To:         e.To,
				Type:       e.Type,
				Confidence: string(e.Confidence),
			})
		}
	}

	return nil, QueryGraphOutput{
		Project: filepath.Base(input.Path),
		Nodes:   nodes,
		Edges:   edges,
		Summary: fmt.Sprintf("Found %d nodes and %d edges from %d starting points", len(nodes), len(edges), len(startNodes)),
	}, nil
}

func (gs *GraphServer) bfs(startNodes []string, depth int, visited map[string]bool, edges *[]*graph.Edge) {
	frontier := make(map[string]bool)
	for _, n := range startNodes {
		frontier[n] = true
		visited[n] = true
	}

	for d := 0; d < depth; d++ {
		nextFrontier := make(map[string]bool)
		for nodeID := range frontier {
			for _, e := range gs.edges {
				var neighbor string
				if e.From == nodeID {
					neighbor = e.To
				} else if e.To == nodeID {
					neighbor = e.From
				} else {
					continue
				}
				if !visited[neighbor] {
					nextFrontier[neighbor] = true
					*edges = append(*edges, e)
				}
			}
		}
		for n := range nextFrontier {
			visited[n] = true
		}
		frontier = nextFrontier
	}
}

func (gs *GraphServer) dfs(startNodes []string, depth int, visited map[string]bool, edges *[]*graph.Edge) {
	type stackItem struct {
		nodeID string
		depth  int
	}
	stack := make([]stackItem, 0, len(startNodes))
	for _, n := range startNodes {
		stack = append(stack, stackItem{n, 0})
	}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[item.nodeID] || item.depth > depth {
			continue
		}
		visited[item.nodeID] = true

		for _, e := range gs.edges {
			var neighbor string
			if e.From == item.nodeID {
				neighbor = e.To
			} else if e.To == item.nodeID {
				neighbor = e.From
			} else {
				continue
			}
			if !visited[neighbor] {
				stack = append(stack, stackItem{neighbor, item.depth + 1})
				*edges = append(*edges, e)
			}
		}
	}
}

func (mgs *MultiGraphServer) getNode(ctx context.Context, req *mcp.CallToolRequest, input GetNodeInput) (*mcp.CallToolResult, GetNodeOutput, error) {
	gs, err := mgs.getGraph(input.Path)
	if err != nil {
		return nil, GetNodeOutput{
			Project: input.Path,
			Error:   err.Error(),
		}, nil
	}

	// Try exact match first
	if n, ok := gs.nodeMap[input.ID]; ok {
		return nil, GetNodeOutput{
			Project: filepath.Base(input.Path),
			Node:    gs.nodeToDetail(n),
		}, nil
	}

	// Try case-insensitive search
	query := strings.ToLower(input.ID)
	for _, n := range gs.nodes {
		if strings.ToLower(n.ID) == query || strings.ToLower(n.Label) == query {
			return nil, GetNodeOutput{
				Project: filepath.Base(input.Path),
				Node:    gs.nodeToDetail(n),
			}, nil
		}
	}

	return nil, GetNodeOutput{
		Project: filepath.Base(input.Path),
		Error:   fmt.Sprintf("Node '%s' not found", input.ID),
	}, nil
}

func (gs *GraphServer) nodeToDetail(n *graph.Node) *NodeDetail {
	inDegree := 0
	outDegree := 0
	for _, e := range gs.edges {
		if e.To == n.ID {
			inDegree++
		}
		if e.From == n.ID {
			outDegree++
		}
	}

	// Find community
	community := -1
	for cid, members := range gs.communities {
		for _, m := range members {
			if m == n.ID {
				community = cid
				break
			}
		}
		if community >= 0 {
			break
		}
	}

	return &NodeDetail{
		ID:        n.ID,
		Label:     n.Label,
		Type:      n.Type,
		Attrs:     n.Attrs,
		InDegree:  inDegree,
		OutDegree: outDegree,
		Community: community,
	}
}

func (mgs *MultiGraphServer) getNeighbors(ctx context.Context, req *mcp.CallToolRequest, input GetNeighborsInput) (*mcp.CallToolResult, GetNeighborsOutput, error) {
	gs, err := mgs.getGraph(input.Path)
	if err != nil {
		return nil, GetNeighborsOutput{
			Project: input.Path,
		}, nil
	}

	if input.Direction == "" {
		input.Direction = "both"
	}

	var neighbors []NeighborInfo
	seen := make(map[string]bool)

	for _, e := range gs.edges {
		var neighborID string
		var direction string

		if e.From == input.ID && (input.Direction == "out" || input.Direction == "both") {
			neighborID = e.To
			direction = "out"
		} else if e.To == input.ID && (input.Direction == "in" || input.Direction == "both") {
			neighborID = e.From
			direction = "in"
		} else {
			continue
		}

		key := neighborID + "-" + e.Type + "-" + direction
		if seen[key] {
			continue
		}
		seen[key] = true

		if n, ok := gs.nodeMap[neighborID]; ok {
			neighbors = append(neighbors, NeighborInfo{
				Node: NodeInfo{
					ID:    n.ID,
					Label: n.Label,
					Type:  n.Type,
				},
				EdgeType:  e.Type,
				Direction: direction,
			})
		}
	}

	return nil, GetNeighborsOutput{
		Project:   filepath.Base(input.Path),
		Neighbors: neighbors,
		Total:     len(neighbors),
	}, nil
}

func (mgs *MultiGraphServer) getCommunity(ctx context.Context, req *mcp.CallToolRequest, input GetCommunityInput) (*mcp.CallToolResult, GetCommunityOutput, error) {
	gs, err := mgs.getGraph(input.Path)
	if err != nil {
		return nil, GetCommunityOutput{
			Project: input.Path,
			Label:   err.Error(),
		}, nil
	}

	members, ok := gs.communities[input.ID]
	if !ok {
		return nil, GetCommunityOutput{
			Project: filepath.Base(input.Path),
			Members: []NodeInfo{},
			Size:    0,
			Label:   fmt.Sprintf("Community %d not found", input.ID),
		}, nil
	}

	var nodeInfos []NodeInfo
	for _, m := range members {
		if n, ok := gs.nodeMap[m]; ok {
			nodeInfos = append(nodeInfos, NodeInfo{
				ID:    n.ID,
				Label: n.Label,
				Type:  n.Type,
			})
		}
	}

	// Generate label from package nodes
	label := fmt.Sprintf("Community %d", input.ID)
	for _, m := range members {
		if n, ok := gs.nodeMap[m]; ok && n.Type == "package" {
			label = n.Label
			break
		}
	}

	return nil, GetCommunityOutput{
		Project: filepath.Base(input.Path),
		Members: nodeInfos,
		Size:    len(members),
		Label:   label,
	}, nil
}

func (mgs *MultiGraphServer) graphSummary(ctx context.Context, req *mcp.CallToolRequest, input GraphSummaryInput) (*mcp.CallToolResult, GraphSummaryOutput, error) {
	gs, err := mgs.getGraph(input.Path)
	if err != nil {
		return nil, GraphSummaryOutput{
			Project: input.Path,
		}, nil
	}

	// Node types
	nodeTypes := make(map[string]int)
	for _, n := range gs.nodes {
		nodeTypes[n.Type]++
	}

	// Edge types and confidence
	edgeTypes := make(map[string]int)
	edgeConf := make(map[string]int)
	for _, e := range gs.edges {
		edgeTypes[e.Type]++
		edgeConf[string(e.Confidence)]++
	}

	// God nodes
	godNodes := analyze.GodNodes(gs.nodes, gs.edges, 5)
	var godNodeInfos []NodeInfo
	for _, g := range godNodes {
		godNodeInfos = append(godNodeInfos, NodeInfo{
			ID:    g.ID,
			Label: g.Label,
			Type:  g.Type,
		})
	}

	// Suggested questions
	questions := analyze.SuggestQuestions(gs.nodes, gs.edges, gs.communities, 3)
	var questionStrs []string
	for _, q := range questions {
		if q.Question != "" {
			questionStrs = append(questionStrs, q.Question)
		}
	}

	return nil, GraphSummaryOutput{
		Project:            filepath.Base(input.Path),
		TotalNodes:         len(gs.nodes),
		TotalEdges:         len(gs.edges),
		NodeTypes:          nodeTypes,
		EdgeTypes:          edgeTypes,
		Communities:        len(gs.communities),
		GodNodes:           godNodeInfos,
		EdgeConfidence:     edgeConf,
		SuggestedQuestions: questionStrs,
	}, nil
}
