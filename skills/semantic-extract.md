# /semantic-extract

Orchestrate LLM semantic extraction for the graphize knowledge graph.

## Overview

This skill automates the full semantic extraction workflow:

1. Get files needing extraction via `graphize enhance --json`
2. Read source files in parallel chunks
3. Dispatch subagents to analyze each chunk
4. Collect results and save to JSON
5. Merge semantic edges into the graph

## Prerequisites

Before running this skill:

- Run `graphize init` to initialize the graph database
- Run `graphize add <repo>` to track source repositories
- Run `graphize analyze` to extract AST-based graph

## Execution Steps

### Step 1: Get Extraction Plan

Run the enhance command with JSON output:

```bash
graphize enhance --json
```

This returns:

```json
{
  "status": "ready",
  "graph_path": "/path/to/.graphize",
  "total_files": 150,
  "cached": 120,
  "uncached": 30,
  "chunk_size": 25,
  "total_chunks": 2,
  "chunks": [
    {
      "id": 1,
      "files": ["file1.go", "file2.go", ...],
      "prompt": "..."
    }
  ]
}
```

If `uncached` is 0, all files are cached and no extraction is needed.

### Step 2: Read Source Files

For each chunk, read all the Go source files using the Read tool.

Read files in parallel for efficiency (up to 10 files per Read batch).

### Step 3: Analyze for Semantic Relationships

For each chunk, analyze the source code to discover semantic relationships NOT captured by AST:

**Relationship Types:**

| Type | Description | Example |
|------|-------------|---------|
| `inferred_depends` | Implicit dependency | Config loaded by multiple handlers |
| `rationale_for` | Design rationale from comments | "// Using X because Y" |
| `similar_to` | Semantic similarity | Two validators with similar logic |
| `implements_pattern` | Design pattern | Service-Repository pattern |
| `shared_concern` | Cross-cutting concern | Error handling, logging |

**Confidence Scoring:**

- 0.8-1.0: Clear evidence in code/comments (INFERRED)
- 0.6-0.8: Strong inference (INFERRED)
- 0.3-0.6: Reasonable guess (INFERRED)
- 0.1-0.3: Uncertain (AMBIGUOUS)

**Node ID Format:**

- Functions: `func_filename.go.FunctionName`
- Methods: `method_ReceiverType.MethodName`
- Types: `type_TypeName`
- Packages: `pkg_packagename`

### Step 4: Generate JSON Output

For each chunk, produce JSON output:

```json
{
  "nodes": [],
  "edges": [
    {
      "from": "func_auth.go.ValidateToken",
      "to": "func_middleware.go.AuthMiddleware",
      "type": "shared_concern",
      "confidence": "INFERRED",
      "confidence_score": 0.85,
      "reason": "Both implement JWT token validation"
    }
  ]
}
```

### Step 5: Save Combined Results

Combine all chunk results and save to `agents/graph/semantic-edges.json`:

```bash
mkdir -p agents/graph
```

Write the combined JSON file with all semantic edges.

### Step 6: Merge into Graph

Validate and merge the semantic edges:

```bash
graphize merge -i agents/graph/semantic-edges.json --validate
```

If validation passes:

```bash
graphize merge -i agents/graph/semantic-edges.json
```

### Step 7: Verify Results

Run report to see the enhanced graph:

```bash
graphize report
```

The report will show:

- Edge confidence breakdown (EXTRACTED vs INFERRED vs AMBIGUOUS)
- New surprising connections from LLM analysis
- Updated community cohesion scores

## Parallel Processing Strategy

For large codebases, dispatch subagents in parallel:

1. Parse `graphize enhance --json` output
2. For each chunk (up to 4 parallel):
   - Read source files
   - Analyze semantics
   - Return JSON edges
3. Collect all results
4. Merge into single JSON file
5. Run `graphize merge`

## Example Session

```
User: /semantic-extract

Claude: Let me orchestrate the semantic extraction.

[Runs graphize enhance --json]

Found 45 files needing extraction in 2 chunks.

[Reads files for chunk 1]
[Reads files for chunk 2]

Analyzing chunk 1/2 (25 files)...
Found 8 semantic relationships.

Analyzing chunk 2/2 (20 files)...
Found 5 semantic relationships.

Total semantic edges discovered: 13
- inferred_depends: 4
- shared_concern: 5
- implements_pattern: 3
- similar_to: 1

Saving to agents/graph/semantic-edges.json...

[Runs graphize merge]

Merged 13 new edges into graph.
New graph totals: 1,250 nodes, 4,532 edges

Run `graphize report` to see the enhanced analysis.
```

## Troubleshooting

**No files to extract:**
All files are cached. Use `graphize enhance --force` to re-analyze.

**Validation errors:**
Check node IDs match graphize convention (type_name format).

**Empty results:**
Some chunks may have no discoverable semantic relationships. This is normal for straightforward code.
