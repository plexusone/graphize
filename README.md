# Graphize

LLM-powered CLI for transforming Go codebases into queryable knowledge graphs.

[![Documentation](https://img.shields.io/badge/docs-plexusone.github.io%2Fgraphize-blue)](https://plexusone.github.io/graphize)

## Features

- 📊 **AST Extraction** - Fast, deterministic extraction of functions, types, and relationships
- 🤖 **LLM Enhancement** - Optional semantic analysis to discover implicit dependencies
- 🔍 **Graph Queries** - BFS/DFS traversal, path finding, community detection
- 📈 **Analysis Reports** - God nodes, surprising connections, suggested questions
- 🌐 **MCP Server** - Integrate with Claude Desktop and Claude Code
- 📤 **Multiple Exports** - HTML visualization, TOON format, JSON, YAML

## Quick Start

```bash
# Build
go build -o graphize ./cmd/graphize

# Initialize a new graph database
graphize init

# Add your Go repository
graphize add .

# Extract the graph (AST-based)
graphize analyze

# Generate an analysis report
graphize report

# Export interactive visualization
graphize export html -o graph.html
```

## Two-Step Extraction

Graphize provides a two-step extraction pipeline:

1. **Deterministic AST extraction** - Fast, reproducible, always available
2. **LLM semantic extraction** - Optional, adds inferred relationships and rationale

```bash
# Step 1: AST extraction
graphize analyze

# Step 2: Prepare for LLM (optional)
graphize enhance --json > files.json

# Step 3: Merge LLM results
graphize merge -i agents/graph/semantic-edges.json
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `graphize init` | Initialize graph database |
| `graphize add <repo>` | Track a repository |
| `graphize status` | Show tracked sources |
| `graphize analyze` | Extract AST-based graph |
| `graphize enhance` | Prepare for LLM extraction |
| `graphize merge` | Merge semantic edges |
| `graphize query` | Query the graph |
| `graphize report` | Generate analysis report |
| `graphize export html` | Cytoscape.js visualization |
| `graphize serve` | Start MCP server |

## MCP Server Integration

Integrate with Claude Desktop or Claude Code:

```json
{
  "mcpServers": {
    "graphize": {
      "command": "graphize",
      "args": ["serve", "-g", "/path/to/.graphize"]
    }
  }
}
```

## Output Formats

| Format | Use Case |
|--------|----------|
| **TOON** | Agent-friendly, token-efficient (default) |
| **JSON** | Machine-readable, full fidelity |
| **YAML** | Human-readable configuration |
| **HTML** | Interactive Cytoscape.js visualization |

## Storage

Uses [GraphFS](https://plexusone.github.io/graphfs) for git-friendly storage:

```
.graphize/
├── manifest.json      # Tracked sources
├── nodes/             # One file per node
├── edges/             # One file per edge
└── cache/             # Per-file extraction cache
```

## Documentation

Full documentation at [plexusone.github.io/graphize](https://plexusone.github.io/graphize)

- [Getting Started](https://plexusone.github.io/graphize/getting-started/)
- [CLI Reference](https://plexusone.github.io/graphize/cli/reference/)
- [MCP Server](https://plexusone.github.io/graphize/mcp-server/)
- [Architecture](https://plexusone.github.io/graphize/architecture/)

## License

MIT
