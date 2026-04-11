# Semantic Extraction

Graphize supports optional LLM semantic extraction to discover relationships not visible in the AST.

## Overview

While AST extraction captures explicit relationships (function calls, imports, type references), semantic extraction uses an LLM to discover implicit relationships:

- Inferred dependencies
- Design patterns
- Shared concerns
- Design rationale

## Two-Step Pipeline

```
┌─────────────────────┐     ┌─────────────────────┐
│ Step 1: AST         │     │ Step 2: Semantic    │
│ (deterministic)     │     │ (LLM-powered)       │
├─────────────────────┤     ├─────────────────────┤
│ • Function calls    │     │ • Inferred depends  │
│ • Imports           │  +  │ • Design patterns   │
│ • Type references   │     │ • Shared concerns   │
│ • Contains edges    │     │ • Rationale         │
└─────────────────────┘     └─────────────────────┘
         │                           │
         └───────────┬───────────────┘
                     ▼
              ┌──────────────┐
              │ Merged Graph │
              │ with confidence │
              │ levels          │
              └──────────────┘
```

## Semantic Edge Types

| Type | Description | Example |
|------|-------------|---------|
| `inferred_depends` | Implicit dependency not in imports | Config values used across packages |
| `rationale_for` | Design rationale from comments | Why a particular pattern was chosen |
| `similar_to` | Semantic similarity | Functions doing similar things |
| `implements_pattern` | Design pattern usage | Factory, Repository, Strategy |
| `shared_concern` | Cross-cutting concern | Logging, authentication, caching |

## Confidence Levels

| Level | Score Range | Meaning |
|-------|-------------|---------|
| `EXTRACTED` | N/A | From AST parsing (deterministic) |
| `INFERRED` | >= 0.3 | LLM-discovered with high confidence |
| `AMBIGUOUS` | < 0.3 | LLM-discovered, needs verification |

## Workflow

### Step 1: Prepare Files

```bash
graphize enhance --json > files.json
```

This outputs files that need semantic analysis, excluding:

- Already-cached files (unchanged since last extraction)
- Generated code
- Test files (optionally)

### Step 2: Run LLM Extraction

Using Claude Code with the `/semantic-extract` skill:

```
/semantic-extract
```

The skill:

1. Reads source files in parallel chunks
2. Analyzes for semantic relationships
3. Assigns confidence scores
4. Outputs to `agents/graph/semantic-edges.json`

### Step 3: Merge Results

```bash
graphize merge -i agents/graph/semantic-edges.json
```

This:

- Adds new semantic edges to the graph
- Sets appropriate confidence levels
- Preserves existing AST-extracted edges

## Manual Extraction

If not using the skill, you can perform extraction manually.

### Prompt Template

For each source file, ask the LLM:

```
Analyze this Go source file for semantic relationships not visible in the AST.

Look for:
1. Implicit dependencies (data flows, shared state)
2. Design patterns (Factory, Repository, Strategy, etc.)
3. Shared concerns (logging, auth, caching)
4. Design rationale in comments

Output JSON:
{
  "edges": [
    {
      "from": "node_id",
      "to": "node_id",
      "type": "inferred_depends|implements_pattern|shared_concern|rationale_for|similar_to",
      "confidence_score": 0.0-1.0,
      "reason": "explanation"
    }
  ]
}

Source file: {filename}
```

### Node ID Convention

Use these ID formats:

| Type | Format | Example |
|------|--------|---------|
| Function | `func_{filename}.{FunctionName}` | `func_handler.go.HandleRequest` |
| Method | `method_{ReceiverType}.{MethodName}` | `method_Service.Process` |
| Type | `type_{TypeName}` | `type_UserService` |
| Package | `pkg_{packagename}` | `pkg_handlers` |
| File | `file_{path}` | `file_pkg/handlers/user.go` |

### Edge Format

```json
{
  "edges": [
    {
      "from": "func_handler.go.HandleRequest",
      "to": "func_db.go.Query",
      "type": "inferred_depends",
      "confidence": "INFERRED",
      "confidence_score": 0.85,
      "reason": "HandleRequest uses query results but doesn't directly call Query"
    }
  ]
}
```

## Caching

Graphize caches extraction results per-file using SHA256 hashes.

```
.graphize/cache/
├── pkg_handlers_user.go.json
├── pkg_handlers_order.go.json
└── ...
```

On re-extraction, only changed files are processed.

## Best Practices

### 1. Extract After AST

Always run `graphize analyze` first to establish the base graph.

### 2. Review AMBIGUOUS Edges

Edges with confidence < 0.3 should be reviewed:

```bash
graphize report | grep AMBIGUOUS
```

### 3. Iterate on Large Codebases

For large codebases, extract incrementally:

1. Start with core packages
2. Review and validate
3. Expand to remaining packages

### 4. Cache Semantic Results

Commit `agents/graph/semantic-edges.json` to preserve LLM work:

```bash
git add agents/graph/semantic-edges.json
git commit -m "chore: update semantic extraction"
```

### 5. Combine with Reports

Use semantic edges to generate richer reports:

```bash
graphize merge -i agents/graph/semantic-edges.json
graphize report -o GRAPH_REPORT.md
```

## Troubleshooting

### No Files to Extract

All files are already cached. Force re-extraction:

```bash
rm -rf .graphize/cache/
graphize enhance --json
```

### Low Confidence Scores

The LLM may be uncertain. Provide more context:

- Include related files in the same prompt
- Add comments explaining design decisions
- Use more specific prompts

### Missing Relationships

Some relationships require cross-file context. Consider:

- Extracting related files together
- Providing package-level context
- Using the MCP server for interactive exploration
