# CLI Overview

Graphize provides a comprehensive CLI for extracting, querying, and exporting knowledge graphs from Go codebases.

## Command Categories

### Setup Commands

| Command | Description |
|---------|-------------|
| `graphize init` | Initialize a new graph database |
| `graphize add <path>` | Add a repository to track |
| `graphize status` | Show tracked sources and their status |

### Extraction Commands

| Command | Description |
|---------|-------------|
| `graphize analyze` | Extract graph from tracked sources (AST) |
| `graphize enhance` | Prepare files for LLM semantic extraction |
| `graphize merge` | Merge semantic edges into the graph |
| `graphize rebuild` | Rebuild graph and merge semantic edges |

### Query Commands

| Command | Description |
|---------|-------------|
| `graphize query` | Query the knowledge graph |
| `graphize diff` | Compare two graph snapshots |

### Analysis Commands

| Command | Description |
|---------|-------------|
| `graphize report` | Generate analysis report |
| `graphize summary` | Generate markdown summary |

### Export Commands

| Command | Description |
|---------|-------------|
| `graphize export html` | Export to interactive HTML (Cytoscape.js) |
| `graphize export toon` | Export to TOON format |
| `graphize export json` | Export to JSON |

### Server Commands

| Command | Description |
|---------|-------------|
| `graphize serve` | Start MCP server for graph queries |

### Agent Commands

| Command | Description |
|---------|-------------|
| `graphize init-agents` | Initialize agent framework directories |

## Global Flags

All commands support these flags:

```bash
-g, --graph string    Path to graph database (default ".graphize")
-f, --format string   Output format: toon, json, yaml (default "toon")
-h, --help            Help for the command
```

## Output Formats

Graphize supports multiple output formats:

### TOON (Default)

Token-optimized notation for AI agents. Compact and efficient.

```bash
graphize query --format toon
```

### JSON

Full-fidelity machine-readable format.

```bash
graphize query --format json
```

### YAML

Human-readable format for configuration and debugging.

```bash
graphize query --format yaml
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GRAPHIZE_PATH` | Default graph database path |
| `GRAPHIZE_FORMAT` | Default output format |

## Next Steps

- [Workflow Guide](workflow.md) - The analyze → enhance → merge flow
- [Command Reference](reference.md) - Detailed command documentation
