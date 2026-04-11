# /graphize enhance

**DEPRECATED**: Use `/semantic-extract` instead for the full automated workflow.

Enhance the graph with LLM-inferred semantic relationships.

## When to Use

Run this skill after `graphize analyze` to add semantic relationships not visible in AST:

- Inferred dependencies between components
- Design rationale from comments
- Semantic similarity between functions/types
- Design pattern implementations

## Quick Start

For automated extraction, use the orchestration skill:

```
/semantic-extract
```

For manual control, continue reading.

## Execution Steps

When this skill is invoked, follow these steps:

### Step 1: Check Prerequisites

```bash
graphize status
```

Verify sources are tracked and analyzed.

### Step 2: Get Files Needing Extraction

```bash
graphize enhance --chunk-size 20
```

This outputs:
- List of files needing semantic extraction (not cached)
- Files grouped into chunks of 20

### Step 3: Read Source Files

For each chunk, read the Go source files listed. Use the Read tool to get file contents.

### Step 4: Analyze Each Chunk

For each chunk of files, analyze them to find semantic relationships:

**Look for:**
1. **inferred_depends**: Packages/functions that work together but aren't explicitly connected
2. **rationale_for**: Comments explaining WHY code was written this way
3. **similar_to**: Functions solving similar problems
4. **implements_pattern**: Factory, Strategy, Observer, Repository patterns
5. **shared_concern**: Error handling, logging, auth that spans files

**Output JSON for each chunk:**
```json
{
  "nodes": [],
  "edges": [
    {
      "from": "func_filename.go.FunctionName",
      "to": "type_TypeName",
      "type": "inferred_depends",
      "confidence": "INFERRED",
      "confidence_score": 0.75,
      "reason": "Both handle user authentication flow"
    }
  ]
}
```

### Step 5: Save Results

After analyzing all chunks, save the semantic edges to a JSON file:

```bash
# Save to agents/graph/semantic-edges.json
```

The edges will be merged with the AST graph on next export.

## Node ID Convention

Node IDs follow this pattern:
- Functions: `func_filename.go.FunctionName`
- Methods: `method_ReceiverType.MethodName`
- Types: `type_TypeName`
- Packages: `pkg_packagename`

## Confidence Guidelines

| Score | Confidence | When to Use |
|-------|------------|-------------|
| 0.8-1.0 | INFERRED | Clear evidence in code/comments |
| 0.6-0.8 | INFERRED | Strong inference from patterns |
| 0.3-0.6 | INFERRED | Reasonable guess |
| 0.1-0.3 | AMBIGUOUS | Uncertain, needs human review |

## Example Output

For a file with authentication code:

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
      "reason": "Both implement JWT token validation logic"
    },
    {
      "from": "type_UserService",
      "to": "type_UserRepository",
      "type": "implements_pattern",
      "confidence": "INFERRED",
      "confidence_score": 0.9,
      "reason": "Service-Repository pattern for user data access"
    }
  ]
}
```

## After Enhancement

Run report to see the new edges:
```bash
graphize report
```

The report will show:
- Edge confidence breakdown (EXTRACTED vs INFERRED vs AMBIGUOUS)
- Surprising connections now include LLM-discovered relationships
