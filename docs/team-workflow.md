# Team Workflow

How to share knowledge graphs across a team using source control.

## Overview

Graphize separates graph extraction into two phases:

| Phase | Cost | Output | Checked In? |
|-------|------|--------|-------------|
| AST extraction | Fast (seconds) | `.graphize/` | No |
| LLM semantic extraction | Expensive (API calls) | `agents/graph/semantic-edges.json` | **Yes** |

This separation allows teams to:

- Run expensive LLM analysis once
- Share semantic edges via git
- Rebuild AST graphs quickly on any machine
- Query via MCP server locally

## Workflow

### Initial Setup (One Time)

One team member runs the full extraction:

```bash
# Initialize and analyze
graphize init
graphize add .
graphize analyze

# Run LLM semantic extraction (expensive)
# This discovers implicit dependencies, design patterns, etc.
graphize enhance --json > /tmp/files.json
# ... run LLM analysis on files ...
# ... save results to agents/graph/semantic-edges.json ...

# Merge semantic edges into graph
graphize merge -i agents/graph/semantic-edges.json

# Generate report for context
graphize report -o agents/graph/GRAPH_REPORT.md
```

### What to Commit

```bash
# Commit semantic edges and reports (small files)
git add agents/graph/semantic-edges.json
git add agents/graph/GRAPH_REPORT.md
git commit -m "chore: add knowledge graph semantic edges"

# Do NOT commit .graphize/ (can be regenerated)
echo ".graphize/" >> .gitignore
```

### On Clone (Fast)

When a team member clones the repo:

```bash
# Re-extract AST graph (fast, deterministic)
graphize init
graphize add .
graphize analyze

# Apply checked-in semantic edges
graphize merge -i agents/graph/semantic-edges.json

# Start MCP server for AI queries
graphize serve
```

This takes seconds, not the minutes/hours of LLM analysis.

## MCP Server Integration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "graphize": {
      "command": "graphize",
      "args": ["serve", "-g", "/path/to/project/.graphize"]
    }
  }
}
```

### Claude Code

Add to `.claude/settings.json` in your project:

```json
{
  "mcpServers": {
    "graphize": {
      "command": "graphize",
      "args": ["serve", "-g", ".graphize"]
    }
  }
}
```

## Keeping Graphs Updated

### After Code Changes

```bash
# Re-analyze (fast)
graphize analyze

# Re-merge semantic edges
graphize merge -i agents/graph/semantic-edges.json

# Optionally regenerate report
graphize report -o agents/graph/GRAPH_REPORT.md
```

### After Semantic Edge Updates

When someone updates the semantic edges:

```bash
git pull
graphize analyze
graphize merge -i agents/graph/semantic-edges.json
```

### Using Watch Mode

For development, auto-rebuild on file changes:

```bash
graphize watch
```

### Using Git Hooks

Auto-rebuild after commits/checkouts:

```bash
graphize hook install
```

## Scaling to Large Codebases

For codebases with 100K+ lines:

### 1. MCP-First Architecture

Don't try to load everything into context. Use MCP queries:

```
# AI agent queries specific things
query_graph("authentication", depth=2)
get_neighbors("func_HandleLogin")
get_community(3)
```

### 2. Chunked Semantic Extraction

Extract semantics per-package or per-module:

```bash
# Extract one package at a time
graphize enhance --package pkg/auth --json
graphize enhance --package pkg/handlers --json
```

### 3. Hierarchical Views

Query at different granularities:

```bash
# Package-level overview
graphize query --types package

# Drill into specific package
graphize query --filter "package:auth"
```

### 4. Filtered Exports

For visualization, export subsets:

```bash
# Only packages
graphize export html --types package -o packages.html

# Only one community
graphize export html --community 3 -o auth.html
```

## File Structure

```
your-project/
├── .gitignore              # Include .graphize/
├── agents/
│   └── graph/
│       ├── semantic-edges.json   # ✓ Checked in
│       └── GRAPH_REPORT.md       # ✓ Checked in
├── .graphize/                    # ✗ Git ignored (regenerated)
│   ├── manifest.json
│   ├── nodes/
│   ├── edges/
│   └── cache/
└── src/
    └── ...
```

## Benefits

| Benefit | Description |
|---------|-------------|
| **Cost efficiency** | LLM analysis runs once, shared by team |
| **Fast setup** | New machines ready in seconds |
| **Version control** | Semantic edges tracked with code |
| **Offline work** | MCP server runs locally |
| **Incremental updates** | Only re-run what changed |

## Comparison to Alternatives

### vs. Checking in full `.graphize/`

| Approach | Git Impact | Clone Time | Flexibility |
|----------|------------|------------|-------------|
| Full `.graphize/` | Large (1000s of files) | Slow | Rigid |
| Semantic edges only | Small (1 file) | Fast | Flexible |

### vs. No persistence

| Approach | Setup Cost | Consistency |
|----------|------------|-------------|
| No persistence | High (re-run LLM) | Variable |
| Semantic edges | Low | Consistent |

## Troubleshooting

### Merge conflicts in semantic-edges.json

Semantic edges are append-only. Resolve by combining both sides:

```bash
git checkout --ours agents/graph/semantic-edges.json
git checkout --theirs agents/graph/semantic-edges-theirs.json
# Merge manually or use graphize merge with both files
```

### Graph out of sync with code

```bash
graphize analyze
graphize merge -i agents/graph/semantic-edges.json
```

### MCP server not finding nodes

Ensure the graph is built:

```bash
graphize status
graphize query  # Should show node/edge counts
```
