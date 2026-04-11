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
graphize analyze [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--no-cache` | Disable caching, re-extract all files | false |
| `--directed` | Treat graph as directed (edges flow from→to) | true |

**Output:**

- Number of nodes extracted
- Number of edges extracted
- Node types breakdown
- Edge types breakdown
- Cache hit/miss statistics

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
| `json` | Cytoscape.js JSON format |
| `graphml` | GraphML XML (for Gephi, yEd) |
| `cypher` | Neo4j Cypher CREATE statements |
| `obsidian` | Wiki-style Obsidian vault |

**Common Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-o, --output` | Output file or directory | stdout |

**Examples:**

```bash
graphize export html -o graph.html
graphize export toon -o GRAPH.toon
graphize export json -o graph.json
graphize export graphml -o graph.graphml
graphize export cypher -o graph.cypher
graphize export obsidian -o ./vault
```

### graphize export cypher

Export graph as Neo4j Cypher CREATE statements.

```bash
graphize export cypher [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-o, --output` | Output file | stdout |

Generates CREATE statements for all nodes and edges, including:

- Node labels and properties
- Edge types and properties
- Confidence metadata for semantic edges

### graphize export obsidian

Export graph as an Obsidian vault with wikilinks.

```bash
graphize export obsidian [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-o, --output` | Output directory | required |
| `--top` | Number of top nodes to include | 20 |
| `--min-degree` | Minimum degree for node pages | 3 |

**Output Structure:**

```
vault/
├── index.md           # Entry point with god nodes
├── communities/       # One page per community
│   ├── community-0.md
│   └── community-1.md
└── nodes/             # One page per significant node
    ├── func_main.md
    └── type_Config.md
```

Pages are interconnected with `[[wikilinks]]` for Obsidian navigation.

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

## graphize path

Find the shortest path between two nodes.

```bash
graphize path <from> <to> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `from` | Starting node ID |
| `to` | Target node ID |

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--max-depth` | Maximum search depth | 10 |

**Examples:**

```bash
graphize path func_main func_handleRequest
graphize path type_Config type_Options --max-depth 5
```

**Output:**

Shows the path with intermediate nodes and edge types:

```
func_main
  --[calls]--> func_init
  --[calls]--> func_handleRequest
```

## graphize benchmark

Show token reduction statistics comparing raw corpus to TOON output.

```bash
graphize benchmark
```

**Output:**

- Raw corpus size (total bytes of source files)
- TOON output size (compressed graph representation)
- Compression ratio
- Token estimates (raw vs TOON)

## graphize watch

Monitor tracked sources for changes and auto-rebuild the graph.

```bash
graphize watch [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--debounce` | Debounce delay for rapid changes | 500ms |
| `--html` | Regenerate HTML visualization on changes | false |
| `--report` | Regenerate analysis report on changes | false |
| `--verbose` | Show detailed file change events | false |

**Examples:**

```bash
# Basic watch mode
graphize watch

# Also regenerate HTML and report
graphize watch --html --report

# Increase debounce for slower systems
graphize watch --debounce 1s
```

Press `Ctrl+C` to stop watching.

## graphize hook

Manage git hooks for automatic graph updates.

### graphize hook install

Install git hooks in the repository.

```bash
graphize hook install
```

Installs:

- `post-commit`: Auto-run `graphize analyze` after commits
- `post-checkout`: Check if graph is stale after checkout

### graphize hook uninstall

Remove graphize git hooks.

```bash
graphize hook uninstall
```

### graphize hook status

Check git hook installation status.

```bash
graphize hook status
```

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
