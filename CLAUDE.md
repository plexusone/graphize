# CLAUDE.md

Instructions for Claude Code when working on Graphize.

## Project Overview

Graphize is an LLM-powered CLI tool that transforms Go codebases into queryable knowledge graphs.

## Build & Test

```bash
# Build
go build ./...

# Test
go test -v ./...

# Run CLI
go run ./cmd/graphize <command>
```

## Available Skills

### /semantic-extract

Orchestrate LLM semantic extraction for the knowledge graph.

**Usage:** Run after `graphize analyze` to add semantic relationships.

**What it does:**

1. Runs `graphize enhance --json` to get files needing extraction
2. Reads source files in parallel chunks
3. Analyzes for semantic relationships (inferred_depends, shared_concern, etc.)
4. Saves results to `agents/graph/semantic-edges.json`
5. Runs `graphize merge` to integrate edges

### /graphize enhance (deprecated)

Use `/semantic-extract` instead for the full automated workflow.

## Key Commands

```bash
graphize init              # Initialize graph database
graphize add <repo>        # Track a repository
graphize status            # Show tracked sources
graphize analyze           # Extract AST-based graph
graphize enhance --json    # Get files for LLM extraction
graphize merge -i <file>   # Merge semantic edges
graphize query             # Query the graph
graphize report            # Generate analysis report
graphize export html       # Cytoscape visualization
graphize serve             # Start MCP server
```

## Semantic Edge Types

When performing LLM extraction, look for these relationship types:

| Type | Description |
|------|-------------|
| `inferred_depends` | Implicit dependency not in imports |
| `rationale_for` | Design rationale from comments |
| `similar_to` | Semantic similarity |
| `implements_pattern` | Design pattern (Factory, Repository, etc.) |
| `shared_concern` | Cross-cutting concern (logging, auth, etc.) |

## Node ID Convention

- Functions: `func_filename.go.FunctionName`
- Methods: `method_ReceiverType.MethodName`
- Types: `type_TypeName`
- Packages: `pkg_packagename`
- Files: `file_path/to/file.go`

## Confidence Levels

- `EXTRACTED`: From AST parsing (deterministic)
- `INFERRED`: LLM-discovered with score >= 0.3
- `AMBIGUOUS`: LLM-discovered with score < 0.3

## Directory Structure

```
.graphize/           # Graph database
  manifest.json      # Tracked sources
  nodes/             # One file per node
  cache/             # Per-file extraction cache
agents/
  graph/             # Generated graph artifacts
  specs/             # multi-agent-spec definitions
skills/
  semantic-extract.md  # Orchestration skill
  enhance.md           # Legacy skill (deprecated)
```
