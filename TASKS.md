# Graphize Tasks

LLM-powered tool to turn Go codebases into queryable knowledge graphs.

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

---

## Feature Comparison: Graphize vs Graphify

| Feature | Graphify | Graphize | Status |
|---------|----------|----------|--------|
| **Source Tracking** | Single directory | Multi-repo with commit hashes | ✅ Better |
| **Git Currency** | None | Tracks commit/branch per repo | ✅ Better |
| **Storage** | Single graph.json | One file per entity (GraphFS) | ✅ Better |
| **AST Extraction** | tree-sitter (20 langs) | Go only (go/ast) | ✅ Done |
| **Per-file Caching** | ✅ SHA256 | ✅ SHA256 | ✅ Done |
| **Edge Confidence** | ✅ | ✅ EXTRACTED/INFERRED/AMBIGUOUS | ✅ Done |
| **LLM Semantic Extraction** | ✅ Subagents | ✅ Skill + merge workflow | ✅ Done |
| **Multi-agent-spec** | ❌ | ✅ agents/specs/ | ✅ Better |
| **Community Detection** | ✅ Leiden/Louvain | ✅ Louvain (gonum) + package-based | ✅ Done |
| **God Nodes Analysis** | ✅ Full | ✅ Full (report cmd) | ✅ Done |
| **Surprising Connections** | ✅ | ✅ Cross-file, cross-community | ✅ Done |
| **Cohesion Scores** | ✅ | ✅ | ✅ Done |
| **Isolated Nodes** | ✅ | ✅ | ✅ Done |
| **Package Statistics** | ❌ | ✅ | ✅ Better |
| **Suggested Questions** | ✅ | ✅ | ✅ Done |
| **Hyperedges** | ✅ | ❌ | ⬜ Low |
| **HTML Visualization** | ✅ vis.js | ✅ cytoscape.js | ✅ Done |
| **TOON Export** | ❌ | ✅ (with confidence) | ✅ Better |
| **JSON Export** | ✅ | ✅ Cytoscape format | ✅ Done |
| **GRAPH_REPORT.md** | ✅ | ✅ (report cmd) | ✅ Done |
| **Obsidian Export** | ✅ | ❌ | ⬜ Low |
| **Neo4j Export** | ✅ | ❌ | ⬜ Low |
| **Watch Mode** | ✅ | ❌ | ⬜ Low |
| **MCP Server** | ✅ | ✅ | ✅ Done |
| **Git Hooks** | ✅ | ❌ | ⬜ Low |

### Graphize Advantages
- **Multi-repo support**: Track multiple repositories with independent git commit tracking
- **GraphFS storage**: Git-friendly one-file-per-entity storage
- **TOON format**: Token-efficient output for AI agents (98% smaller than JSON)
- **multi-agent-spec**: Portable subagent definitions for Claude, Kiro, Codex, Gemini
- **Edge confidence metadata**: Full support for EXTRACTED/INFERRED/AMBIGUOUS with scores

### Graphify Advantages
- **20 language support**: tree-sitter parsers for many languages
- **Community detection**: Leiden/Louvain algorithms built-in
- **Full analysis suite**: God nodes, surprises, questions
- **More export formats**: Obsidian, Neo4j, GraphML

---

## Phase 1 - MVP ✅ COMPLETE

### Core Infrastructure ✅
- [x] Project structure with pkg/ layout
- [x] Source tracking types: Source, Manifest, SourceStatus
- [x] Git commit/branch reading (NewSourceFromPath, CheckStatus)
- [x] Output formatters: JSON, YAML working
- [x] Manifest persistence (save/load to .graphize/manifest.json)
- [ ] TOON output: integrate toon-go library when released

### CLI Commands ✅
- [x] `graphize init` - creates graph database directory structure
- [x] `graphize add <repo>` - tracks repo with commit hash (persisted)
- [x] `graphize status` - shows all sources with staleness detection
- [x] `graphize analyze` - extract graph from sources
- [x] `graphize query` - query the graph with filters

### Go AST Extraction ✅
- [x] Parse Go files with go/ast
- [x] Extract packages, files, functions, methods, types, imports
- [x] Extract function calls, type references as edges
- [x] Mark all AST-derived edges as EXTRACTED confidence

### Query Command ✅
- [x] `graphize query` - show graph summary
- [x] `graphize query <node-id>` - show edges for a node
- [x] BFS/DFS traversal with --depth and --dfs flags
- [x] Direction filter (--dir out/in/both)
- [x] Edge type filter (--edge-type)

### Export Commands ✅
- [x] `graphize export html` - Cytoscape.js visualization
- [x] `graphize export json` - Cytoscape JSON format
- [x] `graphize export toon` - TOON format (agent-optimized)
- [x] `graphize summary` - Markdown summary for AGENTS folder

---

## Phase 2 - LLM Semantic Extraction ✅ COMPLETE

### Per-file Caching ✅
- [x] Create `pkg/cache/cache.go`
- [x] SHA256-based file hashing
- [x] Store extraction results per file hash
- [x] Cache hit/miss detection
- [x] Cache invalidation on file change
- [x] Integrate with `pkg/extract/extract.go`
- [x] Add `--no-cache` flag to analyze command
- [x] Report cache hit/miss statistics

### LLM Extraction Skill
- [x] Create `skills/enhance.md` skill file
- [x] Create `agents/specs/semantic-extractor.yaml` (multi-agent-spec)
- [x] Define subagent prompt for semantic extraction
- [x] Define edge types (inferred_depends, rationale_for, similar_to, etc.)
- [x] Parse and validate subagent JSON output (`pkg/extract/merge.go`)
- [x] Merge with AST extraction results (`graphize merge` command)
- [ ] Create `agents/plugins/` via assistantkit (when available)
- [x] Chunk files (20-25 per chunk) - `extract.ChunkFiles()` + `graphize enhance --json`
- [x] `graphize enhance --prompt` - Output prompts for each chunk
- [x] `graphize enhance --json` - JSON output for automation
- [x] `skills/semantic-extract.md` - Orchestration skill for parallel dispatch

### Edge Confidence System ✅
- [x] Add `Confidence` field to Edge type (EXTRACTED/INFERRED/AMBIGUOUS) - Already in graphfs
- [x] Add `ConfidenceScore` field (0.0-1.0) - Already in graphfs
- [x] AST extraction sets Confidence: EXTRACTED on all edges
- [x] TOON export includes confidence for non-EXTRACTED edges
- [x] HTML export includes confidence data for edge coloring
- [ ] HTML template edge color-coding (in cytoscape-go) - Future enhancement

### New CLI Commands
- [x] `graphize enhance` - Prepare files for LLM semantic extraction
- [x] `graphize enhance --force` - Ignore cache, list all files
- [x] `graphize enhance --chunk-size N` - Control chunk size
- [x] `graphize enhance --prompt` - Output subagent prompts for each chunk
- [x] `graphize enhance --json` - JSON output for automation scripts
- [x] Show cache hit/miss statistics
- [x] `graphize merge -i <json>` - Merge semantic edges from LLM extraction
- [x] `graphize merge --validate` - Validate semantic JSON without merging
- [x] `/semantic-extract` skill - Orchestrated parallel subagent dispatch

### Test Coverage ✅
- [x] `pkg/extract/merge_test.go` - 6 test functions, 31 test cases
  - ChunkFiles, ParseSemanticJSON, ValidateSemanticExtraction
  - MergeExtractions, IsValidSemanticEdgeType, BuildSubagentPrompt
- [x] `cmd/graphize/cmd/enhance_test.go` - 7 test functions
  - EnhanceOutput JSON serialization, ChunkOutput structure
  - Prompt generation, field validation
- [x] `pkg/cache/cache_test.go` - existing cache tests

### Workflow Validation ✅
- [x] End-to-end test on yaml.v2 codebase
- [x] Discovered 18 semantic edges (similar_to, shared_concern, inferred_depends, implements_pattern)
- [x] Merged into graph, verified in report output
- [x] Suggested questions reference INFERRED edges for human review

### Clone & Rebuild Workflow ✅
- [x] `graphize rebuild` - Analyze + merge semantic edges in one command
- [x] `graphize rebuild --html` - Also generate HTML visualization
- [x] `graphize rebuild --report` - Also generate analysis report
- [x] `graphize rebuild --semantics <path>` - Custom semantic edges path
- [x] Automatic detection of `agents/graph/semantic-edges.json`

---

## Phase 3 - Analysis & Reports ✅ COMPLETE

### Community Detection ✅
- [x] Implement simple community detection (pkg/analyze/cluster.go)
- [x] Group by package (natural code communities)
- [x] Connected components fallback
- [x] Calculate cohesion scores
- [x] Split oversized communities
- [x] Community labels inference
- [x] Louvain algorithm via gonum (pkg/analyze/louvain.go)

### Graph Analysis ✅
- [x] God nodes detection (most connected) - pkg/analyze/gods.go
- [x] Surprising connections (cross-community, cross-file) - pkg/analyze/surprise.go
- [x] Isolated nodes detection
- [x] Cross-file edges analysis
- [x] Package statistics
- [x] Edge confidence grouping
- [x] Suggested questions generation (`pkg/analyze/questions.go`)

### Report Generation ✅
- [x] `graphize report` command
- [x] Generate report with:
  - [x] Corpus summary (nodes, edges, types)
  - [x] God nodes listing (most connected)
  - [x] Surprising connections (cross-file, cross-community)
  - [x] Community breakdown with cohesion scores
  - [x] Isolated nodes (potential gaps)
  - [x] Package statistics
  - [x] Edge confidence breakdown
  - [x] INFERRED/AMBIGUOUS edges flagged in Surprising Connections
  - [x] Suggested questions in report (5 question types)

---

## Phase 4 - Agent Integration ✅ COMPLETE

### MCP Server ✅
- [x] `graphize serve` - Start MCP server (using official Go SDK)
- [x] Tool: query_graph (BFS/DFS traversal)
- [x] Tool: get_node (lookup by ID or label)
- [x] Tool: get_neighbors (in/out/both directions)
- [x] Tool: get_community (list members)
- [x] Tool: graph_summary (stats + god nodes + suggested questions)

### AGENTS Folder Convention ✅
- [x] `graphize init-agents` - Create agents/ folder structure
- [x] Generate agents/graph/GRAPH_SUMMARY.md (checkable)
- [x] Generate agents/graph/GRAPH.toon.gz (checkable)
- [x] Add .gitignore for local-only files

---

## Phase 5 - Additional Exports 🔶 IN PROGRESS

### Export Formats
- [x] GraphML export (for Gephi/yEd) - `graphize export graphml`
- [ ] Neo4j Cypher export
- [ ] SVG export
- [ ] Obsidian vault export

### Watch & Hooks
- [ ] `graphize watch` - Monitor files, rebuild on change
- [ ] `graphize hook install` - Git post-commit hook
- [ ] `graphize hook uninstall` - Remove hooks

---

## Current Stats

### Large Codebase (coreforge)
- **21,369 nodes** extracted (AST)
- **72,351 edges** extracted (AST)
- **~20 seconds** extraction time
- **477 KB** TOON export (gzipped)
- **23 MB** HTML visualization

### Semantic Extraction Test (yaml.v2)
- **506 nodes**, **1,738 edges** total
- **1,720 EXTRACTED** edges (AST)
- **18 INFERRED** edges (LLM semantic)
- Edge types discovered: similar_to (5), shared_concern (5), inferred_depends (6), implements_pattern (2)

### Node Types
packages, files, functions, methods, structs, interfaces, constants, variables

### Edge Types
- **AST**: calls, contains, imports, method_of, references, extends
- **Semantic**: inferred_depends, rationale_for, similar_to, implements_pattern, shared_concern

---

## Recommended Workflow

### For Repo Maintainers (Initial Setup)

```bash
# 1. Initialize and extract
graphize init
graphize add .
graphize analyze

# 2. Run LLM semantic extraction (expensive, one-time)
/semantic-extract

# 3. Check in the portable artifacts
git add .graphize/manifest.json
git add agents/graph/semantic-edges.json
git commit -m "feat: add graphize knowledge graph"
```

### For Repo Consumers (After Clone)

```bash
# Single command rebuilds everything
graphize rebuild --html --report

# View the results
open graph.html
```

### What Gets Checked In

| Path | Purpose | Size |
|------|---------|------|
| `.graphize/manifest.json` | Source tracking | <1 KB |
| `agents/graph/semantic-edges.json` | LLM-extracted edges | ~10 KB |
| `agents/specs/` | Subagent definitions | ~5 KB |

### What Gets Generated Locally

| Path | Purpose | Regenerate With |
|------|---------|-----------------|
| `.graphize/nodes/` | Graph database | `graphize rebuild` |
| `.graphize/edges/` | Graph edges | `graphize rebuild` |
| `.graphize/cache/` | Extraction cache | `graphize rebuild` |
| `graph.html` | Visualization | `graphize export html` |
| `GRAPH_REPORT.md` | Analysis | `graphize report` |

---

## Legend

- [x] Implemented
- [ ] Not started
- 🎯 **HIGH** priority
- 🔶 Medium priority
- ⬜ Low priority
