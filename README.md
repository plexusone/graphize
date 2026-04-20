# Graphize

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-pone-svg]][docs-pone-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/graphize/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/graphize/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/graphize/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/graphize/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/graphize/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/graphize/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/graphize
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/graphize
 [docs-pone-svg]: https://img.shields.io/badge/docs-plexusone-blue
 [docs-pone-url]: https://plexusone.github.io/graphize
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/graphize
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/graphize
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fgraphize
 [loc-svg]: https://tokei.rs/b1/github/plexusone/graphize
 [repo-url]: https://github.com/plexusone/graphize
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/graphize/blob/master/LICENSE

LLM-powered CLI for transforming polyglot codebases into queryable knowledge graphs.

## Features

- 🌍 **Multi-Language Support** - Go, Java, TypeScript, and Swift with extensible provider interface
- 📊 **AST Extraction** - Fast, deterministic extraction of functions, types, and relationships
- 🤖 **LLM Enhancement** - Optional semantic analysis to discover implicit dependencies
- 🔍 **Graph Queries** - BFS/DFS traversal, path finding, community detection
- 📈 **Analysis Reports** - God nodes, surprising connections, corpus health, suggested questions
- 💡 **Node Explanation** - Get context with community membership and centrality metrics
- 🌐 **MCP Server** - Integrate with Claude Desktop and Claude Code
- 🔌 **Platform Installers** - One-command setup for Claude, Cursor, Copilot, Codex, Gemini, Aider
- 📤 **Multiple Exports** - HTML, TOON, JSON, GraphML, Neo4j Cypher, Obsidian vault
- 👁️ **Watch Mode** - Auto-rebuild graph on file changes
- 🔗 **Git Hooks** - Automatic analysis on commit/checkout
- 📝 **Doc Extraction** - Link markdown/text documentation to code entities

## Quick Start

```bash
# Build
go build -o graphize ./cmd/graphize

# Initialize a new graph database
graphize init

# Add your repository (Go, Java, TypeScript, Swift)
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
| `graphize path <A> <B>` | Find shortest path between nodes |
| `graphize explain <node>` | Get node context with community and centrality |
| `graphize report` | Generate analysis report |
| `graphize report --health` | Assess corpus health and graph value |
| `graphize benchmark` | Show token reduction statistics |
| `graphize watch` | Auto-rebuild on file changes |
| `graphize hook install` | Install git hooks |
| `graphize install <platform>` | Install AI assistant integrations |
| `graphize export html` | Cytoscape.js visualization |
| `graphize export htmlsite` | Multi-page documentation site |
| `graphize export obsidian` | Obsidian vault with wikilinks |
| `graphize export cypher` | Neo4j Cypher statements |
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
| **HTML** | Interactive Cytoscape.js visualization |
| **HTML Site** | Multi-page documentation site with per-service graphs |
| **GraphML** | Import into Gephi, yEd, Cytoscape desktop |
| **Cypher** | Neo4j CREATE statements |
| **Obsidian** | Wiki-style vault with wikilinks |

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
