# Architecture

Technical architecture of Graphize.

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         GRAPHIZE PIPELINE                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Step 1: Detect        Step 2: Extract           Step 3: Build          │
│  ┌──────────┐          ┌─────────────────┐       ┌──────────────┐       │
│  │ Scan     │          │ Part A: AST     │       │ Merge AST +  │       │
│  │ sources  │─────────▶│ (deterministic) │──┬───▶│ Semantic     │       │
│  │          │          ├─────────────────┤  │    │ results      │       │
│  └──────────┘          │ Part B: LLM     │  │    └──────────────┘       │
│                        │ (optional)      │──┘           │               │
│                        └─────────────────┘              ▼               │
│                                                  ┌──────────────┐       │
│  Step 4: Analyze       Step 5: Export           │ GraphFS      │       │
│  ┌──────────┐          ┌─────────────────┐      │ Store        │       │
│  │ Cluster  │◀─────────│ God nodes       │◀─────└──────────────┘       │
│  │ Detect   │          │ Surprises       │                              │
│  │ (Louvain)│          │ Questions       │                              │
│  └──────────┘          └─────────────────┘                              │
│       │                                                                  │
│       ▼                                                                  │
│  Step 6: Output                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ HTML │ TOON │ JSON │ GRAPH_REPORT.md │                          │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Architecture

```
graphize/
├── cmd/graphize/           # CLI entry point
│   ├── main.go
│   └── cmd/
│       ├── root.go         # Cobra root command
│       ├── init.go         # Initialize database
│       ├── add.go          # Add source repos
│       ├── status.go       # Check source currency
│       ├── analyze.go      # AST extraction
│       ├── enhance.go      # LLM semantic extraction prep
│       ├── merge.go        # Merge semantic edges
│       ├── query.go        # Graph queries
│       ├── report.go       # Analysis reports
│       ├── diff.go         # Graph comparison
│       ├── export.go       # Export (HTML, JSON)
│       ├── serve.go        # MCP server
│       └── summary.go      # Generate summary
│
├── pkg/
│   ├── source/             # Source tracking
│   │   ├── source.go       # Source struct, git integration
│   │   └── manifest.go     # Manifest persistence
│   │
│   ├── extract/            # Extraction engines
│   │   └── ast.go          # Go AST extraction (deterministic)
│   │
│   ├── cache/              # Extraction caching
│   │   └── cache.go        # SHA256-based per-file cache
│   │
│   ├── analyze/            # Graph analysis
│   │   ├── gods.go         # God nodes, isolated nodes
│   │   ├── cluster.go      # Community detection wrapper
│   │   ├── surprise.go     # Surprising connections
│   │   ├── questions.go    # Suggested questions
│   │   └── diff.go         # Graph comparison wrapper
│   │
│   └── output/             # Output formatters
│       ├── output.go       # TOON, JSON, YAML
│       ├── html.go         # Cytoscape.js export
│       └── summary.go      # Markdown summary
│
└── agents/                 # Agent infrastructure
    ├── specs/              # multi-agent-spec definitions
    ├── plugins/            # assistantkit-generated plugins
    └── graph/              # Graph artifacts
        ├── semantic-edges.json
        └── GRAPH_SUMMARY.md
```

## Storage Layer

Graphize uses [GraphFS](https://plexusone.github.io/graphfs) for storage:

```
.graphize/
├── manifest.json           # Tracked sources
├── nodes/
│   ├── func_main.go.Main.json
│   ├── type_UserService.json
│   └── ...
├── edges/
│   ├── {hash}.json
│   └── ...
└── cache/
    ├── pkg_handlers_user.go.json
    └── ...
```

### Node Format

```json
{
  "id": "func_handler.go.HandleRequest",
  "type": "function",
  "label": "HandleRequest",
  "attrs": {
    "source_file": "pkg/handlers/handler.go",
    "package": "handlers",
    "signature": "func HandleRequest(ctx context.Context, req *Request) error"
  }
}
```

### Edge Format

```json
{
  "from": "func_handler.go.HandleRequest",
  "to": "func_db.go.Query",
  "type": "calls",
  "confidence": "EXTRACTED",
  "confidence_score": 1.0
}
```

## Node Types

| Type | Description |
|------|-------------|
| `package` | Go package |
| `file` | Source file |
| `function` | Top-level function |
| `method` | Type method |
| `struct` | Struct type |
| `interface` | Interface type |

## Edge Types

| Type | Confidence | Description |
|------|------------|-------------|
| `contains` | EXTRACTED | Package/file contains entity |
| `imports` | EXTRACTED | Package imports another |
| `calls` | EXTRACTED | Function calls another |
| `implements` | EXTRACTED | Type implements interface |
| `embeds` | EXTRACTED | Type embeds another |
| `inferred_depends` | INFERRED | Implicit dependency |
| `implements_pattern` | INFERRED | Design pattern usage |
| `shared_concern` | INFERRED | Cross-cutting concern |
| `similar_to` | INFERRED | Semantic similarity |
| `rationale_for` | INFERRED | Design rationale |

## Analysis Algorithms

### Community Detection

Uses the Louvain algorithm (via gonum) for modularity optimization:

```go
// From graphfs/pkg/analyze/louvain.go
result := DetectCommunitiesLouvain(nodes, edges, LouvainOptions{
    Resolution: 1.0,
    ExcludeEdgeTypes: []string{"contains", "imports"},
    ExcludeNodeTypes: []string{"package", "file"},
})
```

### Hub Detection

Identifies highly connected nodes by total degree:

```go
// From graphfs/pkg/analyze/gods.go
hubs := FindHubs(nodes, edges, topN, []string{"package", "file"})
```

### Graph Traversal

BFS and DFS traversal for path finding:

```go
// From graphfs/pkg/query/traverse.go
traverser := NewTraverser(graph)
result := traverser.BFS(startNode, Outgoing, maxDepth, edgeTypes)
path := traverser.FindPath(from, to, edgeTypes)
```

## MCP Server

The MCP server (`graphize serve`) exposes tools for AI agents:

| Tool | Purpose |
|------|---------|
| `query_graph` | Search and traverse |
| `get_node` | Node details |
| `get_neighbors` | Adjacent nodes |
| `get_community` | Community members |
| `graph_summary` | Statistics |

See [MCP Server](mcp-server.md) for details.

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/plexusone/graphfs` | Graph storage |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/modelcontextprotocol/go-sdk` | MCP server |
| `gonum.org/v1/gonum` | Graph algorithms |
| `github.com/yaricom/goGraphML` | GraphML export |
| `github.com/grokify/cytoscape-go` | Cytoscape.js export |

## Performance Characteristics

| Operation | Complexity | Typical Time |
|-----------|------------|--------------|
| AST extraction | O(files) | <30s for 20K nodes |
| Community detection | O(edges) | <5s for 70K edges |
| BFS traversal | O(V + E) | <100ms |
| HTML export | O(nodes + edges) | <3s for 20K nodes |

## Design Decisions

### Why GraphFS?

- Git-friendly (one file per entity)
- Deterministic serialization
- Schema validation
- Referential integrity

### Why Louvain?

- Well-understood algorithm
- Available in gonum
- Good balance of quality and speed
- Hierarchical communities

### Why TOON Output?

- ~8x more token-efficient than JSON
- Designed for AI agent consumption
- Preserves essential structure
- Human-readable

### Why Two-Step Extraction?

- AST extraction is deterministic and fast
- LLM extraction is optional and expensive
- Separating them allows incremental updates
- Different confidence levels for different sources
