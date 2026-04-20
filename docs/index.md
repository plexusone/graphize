# Graphize

LLM-powered CLI for transforming polyglot codebases into queryable knowledge graphs.

## Overview

Graphize extracts structure from polyglot codebases and builds queryable knowledge graphs stored in [GraphFS](https://plexusone.github.io/graphfs) format. It combines deterministic AST extraction with optional LLM semantic analysis to create rich, navigable representations of code architecture.

## Supported Languages

| Language | Parser | Framework Detection |
|----------|--------|---------------------|
| Go | Native `go/ast` | - |
| Java | Tree-sitter | Spring (Controller, Service, Repository) |
| TypeScript/JavaScript | Tree-sitter | - |
| Swift | Tree-sitter | - |

External extractors can be added via the [provider interface](architecture.md#provider-interface).

## Features

- **🌍 Multi-Language** - Go, Java, TypeScript, Swift with extensible provider interface
- **📊 AST Extraction** - Fast, deterministic extraction of functions, types, and relationships
- **🤖 LLM Enhancement** - Optional semantic analysis to discover implicit dependencies
- **🔍 Graph Queries** - BFS/DFS traversal, path finding, community detection
- **📈 Analysis Reports** - God nodes, surprising connections, corpus health, suggested questions
- **💡 Node Explanation** - Get context with community membership and centrality metrics
- **🌐 MCP Server** - Integrate with Claude Desktop and Claude Code
- **🔌 Platform Installers** - One-command setup for Claude, Cursor, Copilot, Codex, Gemini, Aider
- **📤 Multiple Exports** - HTML, TOON, JSON, GraphML, Neo4j Cypher, Obsidian vault
- **👁️ Watch Mode** - Auto-rebuild graph on file changes
- **🔗 Git Hooks** - Automatic analysis on commit/checkout
- **📝 Doc Extraction** - Link markdown/text documentation to code entities

## Quick Start

```bash
# Initialize a new graph database
graphize init

# Add your repository (Go, Java, TypeScript, Swift)
graphize add .

# Extract the graph (AST-based)
graphize analyze

# Generate an analysis report
graphize report

# Export interactive visualization
graphize export html
```

## Two-Step Extraction Pipeline

Graphize provides a two-step extraction pipeline:

1. **Deterministic AST extraction** - Fast, reproducible, always available
2. **LLM semantic extraction** - Optional, adds inferred relationships and rationale

```
┌────────────────────────────────────────────────────────────┐
│                      GRAPHIZE PIPELINE                     │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  Step 1: Scan     Step 2: Extract        Step 3: Build     │
│  ┌──────────┐     ┌─────────────────┐    ┌──────────────┐  │
│  │ Detect   │     │ Part A: AST     │    │ Merge AST +  │  │
│  │ sources  │────>│ (deterministic) │─┬─>│ Semantic     │  │
│  │          │     ├─────────────────┤ │  │ results      │  │
│  └──────────┘     │ Part B: LLM     │ │  └──────────────┘  │
│                   │ (optional)      │─┘         │          │
│                   └─────────────────┘           ▼          │
│                                          ┌──────────────┐  │
│  Step 4: Analyze      Step 5: Export     │ GraphFS      │  │
│  ┌──────────┐         ┌─────────────┐    │ Store        │  │
│  │ Cluster  │<────────│ God nodes   │<───└──────────────┘  │
│  │ Detect   │         │ Surprises   │                      │
│  └──────────┘         │ Questions   │                      │
│                       └─────────────┘                      │
└────────────────────────────────────────────────────────────┘
```

## Target Users

- **Developers** exploring unfamiliar codebases
- **AI Agents** (Claude, Codex) needing codebase context
- **Architects** documenting system design
- **Teams** onboarding new members

## Output Formats

| Format | Use Case |
|--------|----------|
| **TOON** | Agent-friendly, token-efficient (default) |
| **JSON** | Machine-readable, full fidelity |
| **HTML** | Interactive Cytoscape.js visualization |
| **GraphML** | Import into Gephi, yEd, Cytoscape desktop |
| **Cypher** | Neo4j CREATE statements |
| **Obsidian** | Wiki-style vault with wikilinks |

## Storage

Graphize stores graphs in [GraphFS](https://plexusone.github.io/graphfs) format:

- One file per node/edge (git-friendly)
- Deterministic JSON serialization
- Schema validation
- Referential integrity

```
.graphize/
├── manifest.json      # Tracked sources
├── nodes/             # One file per node
├── edges/             # One file per edge
└── cache/             # Per-file extraction cache
```

## Next Steps

- [Getting Started](getting-started.md) - Installation and first graph
- [CLI Workflow](cli/workflow.md) - The analyze → enhance → merge flow
- [MCP Server](mcp-server.md) - Claude Desktop/Code integration
- [Architecture](architecture.md) - Technical design details
