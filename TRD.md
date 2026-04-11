# Graphize - Technical Requirements Document

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         GRAPHIZE PIPELINE                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Step 1: Detect        Step 2: Extract           Step 3: Build          │
│  ┌──────────┐          ┌─────────────────┐       ┌──────────────┐       │
│  │ Scan     │          │ Part A: AST     │       │ Merge AST +  │       │
│  │ sources  │─────────▶│ (deterministic) │──┬───▶│ Semantic     │       │
│  │          │          ├─────────────────┤  │    │ results      │       │
│  └──────────┘          │ Part B: LLM     │  │    └──────────────┘       │
│                        │ (optional)      │──┘           │               │
│                        └─────────────────┘              ▼               │
│                                                  ┌──────────────┐       │
│  Step 4: Analyze       Step 5: Export           │ GraphFS      │       │
│  ┌──────────┐          ┌─────────────────┐      │ Store        │       │
│  │ Cluster  │◀─────────│ God nodes       │◀─────└──────────────┘       │
│  │ Detect   │          │ Surprises       │                              │
│  └──────────┘          │ Questions       │                              │
│       │                └─────────────────┘                              │
│       ▼                                                                  │
│  Step 6: Output                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ HTML │ TOON │ JSON │ GRAPH_REPORT.md │ Neo4j │ Obsidian        │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Architecture

```
graphize/
├── cmd/graphize/           # CLI entry point
│   ├── main.go
│   └── cmd/
│       ├── root.go         # Cobra root command
│       ├── init.go         # Initialize database
│       ├── add.go          # Add source repos
│       ├── status.go       # Check source currency
│       ├── analyze.go      # AST extraction
│       ├── enhance.go      # LLM semantic extraction (NEW)
│       ├── query.go        # Graph queries
│       ├── export.go       # Export (HTML, JSON)
│       ├── export_toon.go  # TOON export
│       └── summary.go      # Generate summary
│
├── pkg/
│   ├── source/             # Source tracking
│   │   ├── source.go       # Source struct, git integration
│   │   └── manifest.go     # Manifest persistence
│   │
│   ├── extract/            # Extraction engines
│   │   ├── ast.go          # Go AST extraction (deterministic)
│   │   └── semantic.go     # LLM semantic extraction (NEW)
│   │
│   ├── cache/              # Extraction caching (NEW)
│   │   └── cache.go        # SHA256-based per-file cache
│   │
│   ├── analyze/            # Graph analysis (NEW)
│   │   ├── cluster.go      # Community detection
│   │   ├── gods.go         # God nodes detection
│   │   ├── surprise.go     # Surprising connections
│   │   └── questions.go    # Suggested questions
│   │
│   ├── report/             # Report generation (NEW)
│   │   └── report.go       # GRAPH_REPORT.md
│   │
│   ├── query/              # Query engine
│   │   └── traverse.go     # BFS/DFS traversal
│   │
│   └── output/             # Output formatters
│       └── output.go       # TOON, JSON, YAML
│
└── agents/                 # PlexusOne Agent Infrastructure (NEW)
    ├── specs/              # multi-agent-spec definitions
    │   └── semantic-extractor.yaml
    ├── plugins/            # assistantkit-generated plugins
    │   ├── claude/         # Claude Code plugins
    │   ├── kiro/           # AWS Kiro CLI plugins
    │   ├── codex/          # Codex CLI plugins
    │   └── gemini/         # Gemini CLI plugins
    └── graph/              # Checked-in graph data (optional)
        ├── GRAPH_SUMMARY.md
        └── GRAPH.toon.gz
```

## PlexusOne Agent Infrastructure

Graphize integrates with the PlexusOne agent ecosystem for multi-platform AI agent support.

### Reference Projects

| Project | Purpose | Location |
|---------|---------|----------|
| **multi-agent-spec** | Define subagent specifications | `~/go/src/github.com/plexusone/multi-agent-spec` |
| **assistantkit** | Generate plugins for AI agents | `~/go/src/github.com/plexusone/assistantkit` |
| **agent-team-release** | Reference implementation | `~/go/src/github.com/plexusone/agent-team-release` |

### Directory Conventions

| Project Size | Specs Location | Plugins Location | Graph Location |
|--------------|----------------|------------------|----------------|
| Small (agent-focused) | `specs/` | `plugins/` | N/A |
| Large (app with agents) | `agents/specs/` | `agents/plugins/` | `agents/graph/` |

### Workflow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    AGENT GENERATION WORKFLOW                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. Define spec in multi-agent-spec format                              │
│     agents/specs/semantic-extractor.yaml                                │
│                                                                          │
│  2. Generate plugins via assistantkit                                    │
│     assistantkit generate --spec agents/specs/ --out agents/plugins/    │
│                                                                          │
│  3. Output for each supported AI agent:                                  │
│     ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                   │
│     │ Claude Code │  │ AWS Kiro    │  │ Codex CLI   │  ...              │
│     │ plugin      │  │ plugin      │  │ plugin      │                   │
│     └─────────────┘  └─────────────┘  └─────────────┘                   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Supported AI Agents (via assistantkit)

| Agent | Plugin Format | Status |
|-------|---------------|--------|
| Claude Code | `.md` skill file | ✅ Supported |
| AWS Kiro CLI | Kiro format | ✅ Supported |
| Codex CLI | Codex format | ✅ Supported |
| Gemini CLI | Gemini format | 🔶 In progress |

### Skills Support (Future)

Currently, multi-agent-spec and assistantkit have good support for **subagents**.
To support Claude Code **skills** (and equivalent features in other agents), we should:

1. Extend multi-agent-spec schema to support skill definitions
2. Add skill generation to assistantkit
3. Generate skills for: Claude Code, AWS Kiro, Codex, Gemini CLI, etc.

This would allow graphize to define skills once and generate for all platforms.

## LLM Semantic Extraction Design

### Approach: Multi-Agent Spec with Subagents

The LLM step is defined using **multi-agent-spec** and generated for multiple AI agents via **assistantkit**. This provides:

1. **Multi-platform** - Works with Claude Code, Kiro, Codex, Gemini CLI
2. **Parallel execution** - Multiple file chunks processed via subagents
3. **Caching** - Results cached per-file by SHA256 hash
4. **Incremental** - Only re-extracts changed files
5. **Spec-driven** - Single definition, multiple outputs

### Subagent Spec (multi-agent-spec format)

```yaml
# agents/specs/semantic-extractor.yaml
apiVersion: multi-agent-spec/v1
kind: Subagent
metadata:
  name: semantic-extractor
  description: Extract semantic relationships from Go code
spec:
  input:
    files: list[string]      # File paths to analyze
    chunk_id: int            # Chunk number (for parallel dispatch)
    total_chunks: int        # Total chunks
  output:
    schema:
      nodes: list[Node]
      edges: list[Edge]
  prompt: |
    Analyze the Go files and extract semantic relationships...
```

### Extraction Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    LLM SEMANTIC EXTRACTION                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. Check Cache                                                          │
│     ┌──────────────┐                                                     │
│     │ SHA256 hash  │──▶ If cached & file unchanged, skip                │
│     │ per file     │                                                     │
│     └──────────────┘                                                     │
│                                                                          │
│  2. Chunk Files (20-25 per chunk)                                       │
│     ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐                                     │
│     │ C1  │ │ C2  │ │ C3  │ │ C4  │  ...                                │
│     └─────┘ └─────┘ └─────┘ └─────┘                                     │
│                                                                          │
│  3. Dispatch ALL subagents in SINGLE message (parallel)                 │
│     ┌────────────────────────────────────────────────────────────┐      │
│     │ [Task call 1] [Task call 2] [Task call 3] [Task call 4]    │      │
│     └────────────────────────────────────────────────────────────┘      │
│                          ▼                                               │
│  4. Each subagent extracts:                                              │
│     - Semantic relationships (not visible in AST)                        │
│     - Design rationale from comments                                     │
│     - Cross-file dependencies                                            │
│     - Confidence scores                                                  │
│                          ▼                                               │
│  5. Merge with AST results, save to cache                               │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Edge Confidence System

| Confidence | When Used | Score Range | Source |
|------------|-----------|-------------|--------|
| `EXTRACTED` | Explicit in code (import, call) | 1.0 | AST |
| `INFERRED` | Reasonable inference | 0.6-0.9 | LLM |
| `AMBIGUOUS` | Uncertain, needs review | 0.1-0.3 | LLM |

### Subagent Prompt Structure

```
You are a graphize semantic extraction subagent. Analyze the Go files
and extract relationships not visible in AST.

Files (chunk N of M):
- path/to/file1.go
- path/to/file2.go
...

Extract:
1. INFERRED edges: dependencies implied but not explicit
   - Shared data structures
   - Implicit contracts
   - Architectural patterns

2. Rationale: WHY decisions were made (from comments)
   - Design decisions
   - Trade-offs
   - Constraints

3. Semantic similarity: concepts solving same problem

Output JSON:
{
  "nodes": [...],
  "edges": [
    {
      "from": "node_id",
      "to": "node_id",
      "type": "inferred_depends|rationale_for|similar_to",
      "confidence": "INFERRED|AMBIGUOUS",
      "confidence_score": 0.75,
      "reason": "Why this relationship exists"
    }
  ]
}
```

## Caching Design

```go
// pkg/cache/cache.go

type Cache struct {
    Dir string // .graphize/cache/
}

// Key: SHA256(file_content + absolute_path)
// Value: JSON with nodes/edges extracted from that file

func (c *Cache) Get(path string) (*Extraction, bool)
func (c *Cache) Set(path string, ext *Extraction) error
func (c *Cache) Hash(path string) (string, error)
```

## Community Detection

Using Louvain algorithm (available in Go):

```go
// pkg/analyze/cluster.go

import "github.com/gyuho/goraph"

func DetectCommunities(g *graph.Graph) map[int][]string {
    // Returns: community_id -> []node_ids
}

func CohesionScore(g *graph.Graph, nodes []string) float64 {
    // Ratio of actual to possible intra-community edges
}
```

## CLI Commands

### Existing
- `graphize init` - Initialize database
- `graphize add <repo>` - Add source repository
- `graphize status` - Show source currency
- `graphize analyze` - AST extraction
- `graphize query` - Query graph
- `graphize export html|json|toon` - Export graph
- `graphize summary` - Generate markdown summary

### New Commands
- `graphize enhance` - Run LLM semantic extraction
- `graphize report` - Generate GRAPH_REPORT.md
- `graphize cluster` - Run community detection

## Data Flow

```
Source Code
     │
     ▼
┌─────────────┐     ┌─────────────┐
│ graphize    │────▶│ .graphize/  │
│ analyze     │     │   nodes/    │  (AST extraction - local)
│             │     │   edges/    │
└─────────────┘     └─────────────┘
     │                    │
     ▼                    ▼
┌─────────────┐     ┌─────────────┐
│ graphize    │────▶│ .graphize/  │
│ enhance     │     │   cache/    │  (LLM extraction - cached)
│ (optional)  │     │   nodes/    │
└─────────────┘     │   edges/    │
                    └─────────────┘
                          │
                          ▼
┌─────────────┐     ┌───────────────────────────────────────┐
│ graphize    │────▶│ agents/                               │
│ export      │     │   specs/     (agent definitions)      │
│             │     │   plugins/   (generated for each AI)  │
└─────────────┘     │   graph/     (checked into git)       │
                    │     GRAPH_SUMMARY.md  (~3KB)          │
                    │     GRAPH.toon.gz     (~500KB)        │
                    └───────────────────────────────────────┘
                          │
                          ▼
                    ┌─────────────┐
                    │ graph.html  │  (Generated locally)
                    │ (gitignored)│
                    └─────────────┘
```

### What Gets Checked Into Git

| Path | Contents | Size | Purpose |
|------|----------|------|---------|
| `agents/specs/` | multi-agent-spec YAML | ~2KB | Subagent definitions |
| `agents/plugins/` | Generated plugins | ~10KB | Platform-specific agents |
| `agents/graph/GRAPH_SUMMARY.md` | Markdown summary | ~3KB | Quick context for agents |
| `agents/graph/GRAPH.toon.gz` | Full graph (gzipped) | ~500KB | Complete graph data |
| `.graphize/` | **NOT checked in** | ~50MB | Local working data |
| `graph.html` | **NOT checked in** | ~25MB | Generated on demand |

## Dependencies

### Existing
- `github.com/spf13/cobra` - CLI framework
- `github.com/plexusone/graphfs` - Graph storage
- `github.com/grokify/cytoscape-go` - HTML visualization

### New (for LLM features)
- Community detection: `github.com/gyuho/goraph` or implement Louvain
- No external LLM SDK needed (uses Claude Code's Task tool)

## Performance Targets

| Operation | Target | Current |
|-----------|--------|---------|
| AST extraction (20K nodes) | <30s | ~20s ✅ |
| LLM enhancement (100 files) | <5min | N/A |
| HTML export (20K nodes) | <5s | ~3s ✅ |
| TOON export (20K nodes) | <2s | ~1s ✅ |
| Cache hit | <100ms | N/A |

## Security Considerations

1. **No secrets in graph** - Skip .env, credentials files
2. **Sanitize labels** - Prevent XSS in HTML export
3. **Validate LLM output** - Schema validation on subagent JSON
4. **Local-first** - No data sent to external services (except Claude API via skill)
