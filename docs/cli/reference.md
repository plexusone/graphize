# Command Reference

Complete reference for all Graphize CLI commands.

## graphize init

Initialize a new graph database.

```bash
graphize init
```

Creates the `.graphize/` directory with:

- `manifest.json` - Source tracking
- `nodes/` - Node storage
- `edges/` - Edge storage
- `cache/` - Extraction cache

## graphize add

Add a repository to track.

```bash
graphize add <path>
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `path` | Path to Go repository |

**Examples:**

```bash
graphize add .
graphize add /path/to/repo
graphize add ../sibling-project
```

## graphize status

Show status of tracked sources.

```bash
graphize status
```

**Output:**

- Source path
- Git commit hash
- Git branch
- Last analyzed timestamp
- Currency status (current/stale)

## graphize analyze

Extract graph from tracked sources using AST parsing.

```bash
graphize analyze
```

**Output:**

- Number of nodes extracted
- Number of edges extracted
- Node types breakdown
- Edge types breakdown

All edges have `EXTRACTED` confidence level.

## graphize enhance

Prepare files for LLM semantic extraction.

```bash
graphize enhance [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Output as JSON | false |

**Examples:**

```bash
# Human-readable output
graphize enhance

# JSON for scripting
graphize enhance --json > files.json
```

## graphize merge

Merge semantic edges from LLM extraction into the graph.

```bash
graphize merge [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-i, --input` | Input file with semantic edges | required |

**Examples:**

```bash
graphize merge -i agents/graph/semantic-edges.json
```

**Input Format:**

```json
{
  "edges": [
    {
      "from": "func_handler.go.HandleRequest",
      "to": "func_db.go.Query",
      "type": "inferred_depends",
      "confidence": "INFERRED",
      "confidence_score": 0.85
    }
  ]
}
```

## graphize rebuild

Rebuild graph from sources and merge semantic edges.

```bash
graphize rebuild
```

Equivalent to:

```bash
graphize analyze && graphize merge -i agents/graph/semantic-edges.json
```

## graphize query

Query the knowledge graph.

```bash
graphize query [node-id] [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `node-id` | Node ID to query (optional) |

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--from` | Filter edges by source node | |
| `--type` | Filter edges by type | |
| `--limit` | Maximum results | 100 |
| `--depth` | Traversal depth (enables BFS/DFS) | 0 |
| `--dfs` | Use depth-first search | false |
| `--dir` | Direction: out, in, both | both |
| `--path` | Find path to this node | |
| `--edge-type` | Filter by edge type(s), comma-separated | |

**Examples:**

```bash
# Show graph summary
graphize query

# Show edges for a node
graphize query func_main

# BFS traverse 3 levels deep
graphize query func_main --depth 3

# DFS traverse
graphize query func_main --dfs --depth 5

# Only outgoing edges
graphize query func_main --dir out

# Only incoming edges
graphize query func_main --dir in

# Find path between nodes
graphize query func_main --path func_handleRequest

# Filter by edge type
graphize query --type calls
```

## graphize diff

Compare two graph snapshots.

```bash
graphize diff [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--old` | Path to old graph | required |

**Examples:**

```bash
graphize diff --old .graphize.bak
graphize diff --old /path/to/old/graph --graph /path/to/new/graph
```

**Output:**

- New nodes added
- Nodes removed
- New edges added
- Edges removed

## graphize report

Generate analysis report for the graph.

```bash
graphize report [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--top` | Number of top items to show | 10 |
| `-o, --output` | Output file | stdout |

**Examples:**

```bash
graphize report
graphize report --top 20 -o GRAPH_REPORT.md
```

**Report Sections:**

- Summary (nodes, edges, types)
- God Nodes (most connected)
- Communities (Louvain detection)
- Surprising Connections
- Isolated Nodes
- Package Statistics
- Suggested Questions

## graphize summary

Generate a markdown summary of the graph.

```bash
graphize summary [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-o, --output` | Output file | stdout |

## graphize export

Export graph to various formats.

```bash
graphize export <format> [flags]
```

**Formats:**

| Format | Description |
|--------|-------------|
| `html` | Interactive Cytoscape.js visualization |
| `toon` | Token-optimized notation |
| `json` | Full JSON export |

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-o, --output` | Output file | stdout |

**Examples:**

```bash
graphize export html -o graph.html
graphize export toon -o GRAPH.toon
graphize export json -o graph.json
```

## graphize serve

Start MCP server for graph queries.

```bash
graphize serve
```

Starts a Model Context Protocol server over stdio.

**Available Tools:**

| Tool | Description |
|------|-------------|
| `query_graph` | Search and traverse the graph |
| `get_node` | Get details for a specific node |
| `get_neighbors` | Get neighbors of a node |
| `get_community` | Get nodes in a community |
| `graph_summary` | Get overall graph statistics |

See [MCP Server](../mcp-server.md) for integration details.

## graphize init-agents

Initialize agent framework directories.

```bash
graphize init-agents
```

Creates:

```
agents/
├── specs/       # multi-agent-spec definitions
├── plugins/     # assistantkit-generated plugins
└── graph/       # Graph artifacts
```

## graphize completion

Generate shell completion scripts.

```bash
graphize completion <shell>
```

**Shells:**

- `bash`
- `zsh`
- `fish`
- `powershell`

**Examples:**

```bash
# Bash
graphize completion bash > /etc/bash_completion.d/graphize

# Zsh
graphize completion zsh > "${fpath[1]}/_graphize"

# Fish
graphize completion fish > ~/.config/fish/completions/graphize.fish
```
