# CLI Workflow

This guide explains the typical Graphize workflow from initialization to visualization.

## Basic Workflow

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  init    │───▶│   add    │───▶│ analyze  │───▶│  query   │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
                                      │
                                      ▼
                               ┌──────────┐
                               │  export  │
                               └──────────┘
```

### Step 1: Initialize

```bash
graphize init
```

Creates the `.graphize/` directory structure.

### Step 2: Add Sources

```bash
graphize add .
```

Tracks the current directory as a source. You can add multiple repositories:

```bash
graphize add /path/to/repo1
graphize add /path/to/repo2
```

### Step 3: Analyze

```bash
graphize analyze
```

Extracts the AST-based graph. This is deterministic and fast.

### Step 4: Query or Export

```bash
# Interactive queries
graphize query func_main --depth 3

# Generate report
graphize report

# Export visualization
graphize export html -o graph.html
```

## Enhanced Workflow (with LLM)

```
┌──────────┐    ┌──────────┐    ┌──────────┐
│ analyze  │───▶│ enhance  │───▶│  merge   │
└──────────┘    └──────────┘    └──────────┘
                     │
                     ▼
              ┌─────────────┐
              │ LLM Agent   │
              │ (external)  │
              └─────────────┘
```

### Step 1: Analyze (AST)

```bash
graphize analyze
```

### Step 2: Enhance (Prepare for LLM)

```bash
graphize enhance --json > files-to-analyze.json
```

This outputs a list of files that need semantic analysis.

### Step 3: Run LLM Extraction

Using Claude Code with the `/semantic-extract` skill:

```bash
# In Claude Code
/semantic-extract
```

This:

1. Reads the source files
2. Analyzes for semantic relationships
3. Outputs `agents/graph/semantic-edges.json`

### Step 4: Merge Results

```bash
graphize merge -i agents/graph/semantic-edges.json
```

Integrates the semantic edges into the graph with appropriate confidence levels.

### Shortcut: Rebuild

If you've already done semantic extraction once:

```bash
graphize rebuild
```

This combines `analyze` + `merge` in one step.

## Query Patterns

### Graph Summary

```bash
graphize query
```

Shows node/edge counts, types, and top connected nodes.

### Node Exploration

```bash
# Show edges for a node
graphize query func_main

# BFS traversal
graphize query func_main --depth 3

# DFS traversal
graphize query func_main --depth 3 --dfs

# Direction filtering
graphize query func_main --dir out   # What does it call?
graphize query func_main --dir in    # What calls it?
```

### Path Finding

```bash
graphize query --path func_main func_handleRequest
```

### Edge Type Filtering

```bash
graphize query func_main --edge-type calls,imports
```

## Export Patterns

### Interactive HTML

```bash
graphize export html -o graph.html
```

Opens a Cytoscape.js visualization with:

- Pan and zoom
- Node filtering by type
- Edge filtering by confidence
- Search functionality

### TOON for Agents

```bash
graphize export toon -o GRAPH.toon
gzip GRAPH.toon  # Compress for storage
```

### JSON for Processing

```bash
graphize export json -o graph.json
```

## Analysis Patterns

### Full Report

```bash
graphize report -o GRAPH_REPORT.md
```

Generates:

- Summary statistics
- God nodes (most connected)
- Community detection results
- Surprising connections
- Suggested questions
- Isolated nodes
- Package statistics

### Quick Summary

```bash
graphize summary
```

Shorter output focused on key metrics.

### Graph Comparison

```bash
# After making changes
graphize analyze

# Compare with previous snapshot
graphize diff --old .graphize.bak
```

## Best Practices

### 1. Commit the Cache

The `.graphize/cache/` directory stores per-file extraction results. Committing it speeds up re-analysis after minor changes.

### 2. Exclude Large Generated Files

Add patterns to your source tracking to exclude generated code:

```bash
# Edit .graphize/manifest.json to add excludes
```

### 3. Use TOON for AI Agents

TOON format is ~8x more token-efficient than JSON:

```bash
graphize export toon -o agents/graph/GRAPH.toon
```

### 4. Incremental Updates

After code changes:

```bash
graphize status           # Check what changed
graphize analyze          # Re-extract (uses cache)
graphize report           # Update analysis
```

### 5. MCP Server for Interactive Use

For ongoing queries during development:

```bash
graphize serve
```

Then use Claude Desktop/Code to query the graph conversationally.
