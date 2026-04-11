# Getting Started

This guide walks you through installing Graphize and creating your first knowledge graph.

## Prerequisites

- Go 1.24 or later
- A Go codebase to analyze

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/plexusone/graphize.git
cd graphize

# Build
go build -o graphize ./cmd/graphize

# Install to PATH (optional)
go install ./cmd/graphize
```

## Quick Start

### 1. Initialize the Graph Database

Navigate to your Go project and initialize Graphize:

```bash
cd /path/to/your/go/project
graphize init
```

This creates a `.graphize/` directory to store the graph data.

### 2. Add Source Repository

Add your codebase as a tracked source:

```bash
# Add current directory
graphize add .

# Or add a specific path
graphize add /path/to/another/repo
```

### 3. Extract the Graph

Run AST extraction to build the knowledge graph:

```bash
graphize analyze
```

This extracts:

- Packages, files, functions, methods, types
- Calls, imports, contains relationships
- All with `EXTRACTED` confidence level

### 4. Check Status

View tracked sources and their status:

```bash
graphize status
```

### 5. Query the Graph

Explore the extracted graph:

```bash
# Show graph summary
graphize query

# Query a specific node
graphize query func_main

# Traverse from a node (BFS)
graphize query func_main --depth 3

# Find path between nodes
graphize query --path func_main func_handleRequest
```

### 6. Generate Report

Create an analysis report with insights:

```bash
graphize report
```

The report includes:

- Node and edge statistics
- God nodes (most connected entities)
- Community detection results
- Surprising connections
- Suggested questions

### 7. Export Visualization

Generate an interactive HTML visualization:

```bash
graphize export html -o graph.html
```

Open `graph.html` in your browser to explore the graph visually.

## Next Steps

### Add Semantic Extraction (Optional)

Enhance the graph with LLM-inferred relationships:

```bash
# Prepare files for LLM extraction
graphize enhance --json > files.json

# Run semantic extraction (requires Claude Code)
# See: Semantic Extraction guide

# Merge semantic edges
graphize merge -i agents/graph/semantic-edges.json
```

### Start MCP Server

Integrate with Claude Desktop or Claude Code:

```bash
graphize serve
```

See the [MCP Server](mcp-server.md) guide for configuration.

### Enable Watch Mode

Auto-rebuild the graph when files change:

```bash
graphize watch
```

### Install Git Hooks

Automatically analyze on commits:

```bash
graphize hook install
```

## Directory Structure

After initialization and analysis:

```
your-project/
├── .graphize/           # Graph database
│   ├── manifest.json    # Tracked sources
│   ├── nodes/           # One file per node
│   ├── edges/           # One file per edge
│   └── cache/           # Per-file extraction cache
├── agents/              # Agent artifacts (optional)
│   └── graph/
│       ├── semantic-edges.json
│       └── GRAPH_SUMMARY.md
└── ... your source files
```

## Common Options

All commands support these global flags:

| Flag | Description | Default |
|------|-------------|---------|
| `-g, --graph` | Path to graph database | `.graphize` |
| `-f, --format` | Output format: toon, json, yaml | `toon` |

## Troubleshooting

### "No nodes found"

Run `graphize analyze` first to extract the graph.

### "Node not found"

Use `graphize query` without arguments to see available nodes, then search:

```bash
graphize query | grep -i "function_name"
```

### Large graphs are slow

For very large codebases:

1. Use `--limit` flag to restrict results
2. Export to HTML and use the visualization filters
3. Focus queries on specific packages or functions
