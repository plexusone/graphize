# Graphize Tasks

LLM-powered tool to turn Go codebases into queryable knowledge graphs.

## Feature Parity Reference

Reviewed against [safishamsi/graphify](https://github.com/safishamsi/graphify):

- **Branch:** `v3`
- **Commit:** `699e9960ce7b88076db33a4da3adbd53fb410c7c`
- **Version:** v0.3.28+5 (2026-04-10)
- **Review Date:** 2026-04-11

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

### Core Features

| Feature | Graphify | Graphize | Status |
|---------|----------|----------|--------|
| **Source Tracking** | Single directory | Multi-repo with commit hashes | ✅ Better |
| **Git Currency** | None | Tracks commit/branch per repo | ✅ Better |
| **Storage** | Single graph.json | One file per entity (GraphFS) | ✅ Better |
| **AST Extraction** | tree-sitter (20 langs) | Go only (go/ast) | ✅ Done |
| **Per-file Caching** | ✅ SHA256 + MD5 | ✅ SHA256 | ✅ Done |
| **Edge Confidence** | ✅ | ✅ EXTRACTED/INFERRED/AMBIGUOUS | ✅ Done |
| **LLM Semantic Extraction** | ✅ Subagents | ✅ Skill + merge workflow | ✅ Done |
| **Multi-agent-spec** | ❌ | ✅ agents/specs/ | ✅ Better |

### Analysis Features

| Feature | Graphify | Graphize | Status |
|---------|----------|----------|--------|
| **Community Detection** | ✅ Leiden/Louvain | ✅ Louvain (gonum) | ✅ Done |
| **God Nodes Analysis** | ✅ Full | ✅ Full (report cmd) | ✅ Done |
| **Surprising Connections** | ✅ Betweenness centrality | ✅ Cross-file, cross-community | ✅ Done |
| **Cohesion Scores** | ✅ | ✅ | ✅ Done |
| **Isolated Nodes** | ✅ | ✅ | ✅ Done |
| **Package Statistics** | ❌ | ✅ | ✅ Better |
| **Suggested Questions** | ✅ | ✅ | ✅ Done |
| **Hyperedges** | ✅ 3+ node groups | ❌ | ⬜ Phase 7 |
| **Betweenness Centrality** | ✅ For bridges | ✅ Bridges in report | ✅ Done |
| **Corpus Health Check** | ✅ Word count, verdict | ❌ | ⬜ Phase 6 |

### Export Formats

| Feature | Graphify | Graphize | Status |
|---------|----------|----------|--------|
| **HTML Visualization** | ✅ vis.js | ✅ cytoscape.js | ✅ Done |
| **TOON Export** | ❌ | ✅ (with confidence) | ✅ Better |
| **JSON Export** | ✅ NetworkX | ✅ Cytoscape format | ✅ Done |
| **GRAPH_REPORT.md** | ✅ | ✅ (report cmd) | ✅ Done |
| **GraphML Export** | ✅ | ✅ | ✅ Done |
| **Obsidian Export** | ✅ Wiki-style vault | ✅ | ✅ Done |
| **Neo4j Cypher Export** | ✅ cypher.txt | ✅ | ✅ Done |
| **Neo4j Push** | ✅ Direct bolt connection | ❌ | ⬜ Phase 5 |
| **SVG Export** | ✅ | ❌ | ⬜ Phase 5 |

### CLI Features

| Feature | Graphify | Graphize | Status |
|---------|----------|----------|--------|
| **MCP Server** | ✅ | ✅ | ✅ Done |
| **Watch Mode** | ✅ fsnotify | ✅ | ✅ Done |
| **Git Hooks** | ✅ post-commit/checkout | ✅ | ✅ Done |
| **Directed Graphs** | ✅ `--directed` flag | ✅ | ✅ Done |
| **Path Command** | ✅ `path "A" "B"` | ✅ | ✅ Done |
| **Explain Command** | ✅ `explain "Node"` | ❌ | ⬜ Phase 6 |
| **Token Benchmark** | ✅ `benchmark` | ✅ | ✅ Done |
| **URL Ingestion** | ✅ `add <url>` | ❌ | ⬜ Phase 7 |

### Content Types

| Feature | Graphify | Graphize | Status |
|---------|----------|----------|--------|
| **Go Code** | ✅ tree-sitter | ✅ go/ast | ✅ Done |
| **Multi-language** | ✅ 20 langs | ❌ Go only | ⬜ Phase 7 |
| **Markdown/Text** | ✅ Claude extraction | ❌ | ⬜ Phase 6 |
| **PDF Papers** | ✅ Citation mining | ❌ | ⬜ Phase 7 |
| **Images** | ✅ Claude vision | ❌ | ⬜ Phase 7 |
| **Video/Audio** | ✅ Whisper transcription | ❌ | ⬜ Phase 7 |
| **Office Docs** | ✅ DOCX/XLSX conversion | ❌ | ⬜ Phase 7 |

### Platform Integration

| Feature | Graphify | Graphize | Status |
|---------|----------|----------|--------|
| **Claude Code** | ✅ PreToolUse hook | ✅ MCP server | ✅ Done |
| **Codex** | ✅ hooks.json | ❌ | ⬜ Phase 6 |
| **Cursor** | ✅ .cursor/rules | ❌ | ⬜ Phase 6 |
| **Gemini CLI** | ✅ BeforeTool hook | ❌ | ⬜ Phase 6 |
| **GitHub Copilot** | ✅ skills/ folder | ❌ | ⬜ Phase 6 |
| **Aider/OpenClaw** | ✅ AGENTS.md | ❌ | ⬜ Phase 6 |

### Graphize Advantages

- **Multi-repo support**: Track multiple repositories with independent git commit tracking
- **GraphFS storage**: Git-friendly one-file-per-entity storage
- **TOON format**: Token-efficient output for AI agents (98% smaller than JSON)
- **multi-agent-spec**: Portable subagent definitions for Claude, Kiro, Codex, Gemini
- **Edge confidence metadata**: Full support for EXTRACTED/INFERRED/AMBIGUOUS with scores

### Graphify Advantages

- **20 language support**: tree-sitter parsers for many languages
- **Multimodal extraction**: Code, docs, papers, images, video, audio, office docs
- **Platform hooks**: 10 AI assistant integrations with always-on hooks
- **URL ingestion**: Fetch and extract papers, tweets, videos

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

## Phase 5 - Export & Automation ✅ COMPLETE

**Implementation Order:** Quick wins first, then multi-language (Phase 7).
See PLAN.md for detailed schedule.

### Quick Wins ✅

- [x] `graphize path "A" "B"` - Trace exact path between nodes ✅
  - [x] Use graphfs query.FindPath
  - [x] Show intermediate nodes and edge types
- [x] `graphize benchmark` - Print token reduction stats ✅
  - [x] Compare raw corpus size vs TOON output
  - [x] Show compression ratio
- [x] `--directed` flag for `graphize analyze` ✅
  - [x] Preserve edge direction in graph (stored in manifest)
  - [x] Affects traversal and analysis

### Export Formats ✅

- [x] GraphML export (for Gephi/yEd) - `graphize export graphml`
- [x] Neo4j Cypher export - `graphize export cypher` ✅
  - [x] Generate CREATE statements for nodes
  - [x] Generate CREATE statements for edges
  - [x] Include all node/edge attributes
- [ ] Neo4j Push - `graphize export cypher --push bolt://localhost:7687`
  - [ ] Direct bolt connection to Neo4j instance
  - [ ] Authentication support (user/password)
- [ ] SVG export - `graphize export svg`
  - [ ] Use gonum/plot or similar for layout
  - [ ] Static vector graph output
- [x] Obsidian vault export - `graphize export obsidian` ✅
  - [x] Generate `index.md` entry point
  - [x] One article per community with wikilinks
  - [x] One article per god node
  - [x] Cohesion scores and navigation footers

### Watch Mode ✅

- [x] `graphize watch` - Monitor files, rebuild on change ✅
  - [x] Use fsnotify for file system events
  - [x] Debounce rapid changes (500ms)
  - [x] Incremental rebuild (only changed files)
  - [x] Optional: auto-regenerate HTML/report

### Git Hooks ✅

- [x] `graphize hook install` - Install git hooks ✅
  - [x] post-commit hook: auto-analyze on commit
  - [x] post-checkout hook: check if graph is stale
- [x] `graphize hook uninstall` - Remove hooks ✅
- [x] `graphize hook status` - Check hook installation ✅

---

## Phase 6 - Enhanced Analysis 🔶 PLANNED

### Analysis Improvements

- [ ] Betweenness centrality for bridge detection
  - [ ] Identify critical path nodes
  - [ ] Use in surprising connections scoring
- [ ] Composite surprise scoring
  - [ ] Weight cross-file > cross-community
  - [ ] Weight code-doc edges higher
- [ ] Corpus health check
  - [ ] File count, word count statistics
  - [ ] Verdict on whether graph adds value
  - [ ] Token reduction percentage

### New Commands

- [ ] `graphize explain "NodeName"` - Explain a node in context
  - [ ] Show node attributes
  - [ ] Show immediate neighbors
  - [ ] Show community membership
  - [ ] Summarize relationships

### Documentation Extraction

- [ ] Markdown/text extraction via LLM
  - [ ] Extract concepts and relationships
  - [ ] Link to code nodes
  - [ ] Support for README, docs/ folders

### Platform Installers

- [ ] `graphize install claude` - Install Claude Code integration
  - [ ] Add PreToolUse hook to settings.json
  - [ ] Add graphize section to CLAUDE.md
- [ ] `graphize install codex` - Codex hooks.json
- [ ] `graphize install cursor` - .cursor/rules/graphize.mdc
- [ ] `graphize install gemini` - Gemini CLI BeforeTool hook
- [ ] `graphize install copilot` - GitHub Copilot skills folder
- [ ] `graphize install aider` - AGENTS.md section

---

## Phase 7 - Multimodal & Multi-language ⬜ FUTURE

### Multi-language Support

- [ ] Tree-sitter integration via go-tree-sitter
  - [ ] Python extraction
  - [ ] TypeScript/JavaScript extraction
  - [ ] Rust extraction
  - [ ] Java extraction
- [ ] Language detection heuristics
- [ ] Unified node ID scheme across languages

### Hyperedges

- [ ] Support for 3+ node group relationships
  - [ ] "All classes implementing interface X"
  - [ ] "All functions in auth flow"
- [ ] Hyperedge visualization in HTML export
- [ ] Hyperedge queries

### URL Ingestion

- [ ] `graphize add <url>` - Fetch and extract external content
  - [ ] Papers (PDF): citation mining + concept extraction
  - [ ] Web pages: content extraction
  - [ ] Videos (YouTube): transcript extraction
- [ ] `--author` and `--contributor` tags
- [ ] URL caching by hash

### Multimodal Extraction

- [ ] PDF extraction
  - [ ] Text extraction via pdftotext or similar
  - [ ] LLM concept extraction
  - [ ] Citation relationship mining
- [ ] Image extraction
  - [ ] Claude vision for diagrams, screenshots, charts
  - [ ] Node creation for visual concepts
- [ ] Video/Audio transcription
  - [ ] Whisper integration for local transcription
  - [ ] God-node-aware prompts for domain vocabulary
  - [ ] Transcript caching

### Office Documents

- [ ] DOCX extraction
  - [ ] Convert to markdown
  - [ ] LLM concept extraction
- [ ] XLSX extraction
  - [ ] Convert to markdown tables
  - [ ] Extract data relationships

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

## Code Quality & Refactoring 🔧 IN PROGRESS

Move business logic from cmd/ to library packages for better unit testability.

### DRY Fixes (Quick Wins)

- [x] Extract `formatBytes()` to `pkg/metrics/formatter.go`
  - Duplicated in: benchmark.go, export_cypher.go, export_graphml.go
  - Added: `FormatBytes()`, `FormatNumber()` with tests
- [x] Extract edge grouping helpers to `pkg/analyze/group.go`
  - `groupEdgesByType()` duplicated in: export_obsidian.go, report.go
  - Added: `GroupEdgesByType()`, `CountEdgesByType()`, `GroupEdgesByConfidence()`, `CountEdgesByConfidence()` with tests
- [x] Extract directory walking to `pkg/metrics/walker.go`
  - File collection duplicated in: benchmark.go, enhance.go
  - Added: `WalkSourceFiles()`, `WalkSourceFilesWithContent()`, `WalkOptions` with tests

### High Priority Extractions

- [x] `pkg/exporters/cypher/` - Cypher generation from export_cypher.go
  - Generator type with Generate(), NodeToCreate(), EdgeToCreate()
  - EscapeString(), EscapeKey(), ToNeoLabel(), ToNeoRelType()
  - Comprehensive unit tests (8 test functions)
- [x] `pkg/metrics/tokens.go` - Token estimation from benchmark.go
  - EstimateTokens(), EstimateTokensInFile()
  - Word counting + punctuation heuristics for LLM token estimation
- [x] `pkg/exporters/obsidian/` - Vault generation from export_obsidian.go
  - Generator type with Generate(), VaultContent result type
  - GenerateIndex(), GenerateCommunity(), GenerateNode()
  - SanitizeName() for safe filenames
  - Comprehensive unit tests (6 test functions)
- [x] `pkg/query/` - Query formatting from query.go ✅
  - [x] `summary.go` - ComputeSummary, SummaryOptions, GodNode detection
  - [x] `format.go` - FormatTraversal, FormatPath for output formatting
  - [x] `filter.go` - EdgeFilter, FilterEdges, FindPartialMatches
  - [x] Unit tests for all functions

### Medium Priority Extractions

- [x] `pkg/analyze/report.go` - Report orchestration from report.go ✅
  - [x] Report struct with all analysis sections
  - [x] GenerateReport() function
  - [x] FormatMarkdown() method
  - [x] Unit tests for report generation
- [x] `pkg/exporters/graphml/` - GraphML generation from export_graphml.go ✅
  - [x] Generator type with Generate(), WriteTo()
  - [x] Result type with NodeCount, EdgeCount, SkippedEdges
  - [x] Configurable: Directed, GraphID, Description
  - [x] Unit tests (9 test functions)

---

## Provider Integrations 🔶 IN PROGRESS

### system-spec Provider ✅
- [x] Register system-spec provider in graphize
  - [x] Import `github.com/plexusone/system-spec/graphize` package
  - [x] Add to provider registry in `pkg/extract/systemspec/extractor.go`
  - [x] Auto-detect system-spec JSON files (has `name` + `services` fields)
  - [x] Extract infrastructure topology nodes alongside code graph
  - [x] Link service nodes to repo paths via `links_to` edges

### Service-Based Filtering
- [ ] Add `--service` flag to query command
  - [ ] Map service name → repo URL via `links_to` edges
  - [ ] Resolve repo URL → local path via manifest
  - [ ] Filter query results to nodes from that repo
  - [ ] Example: `graphize query --service payments` returns all code from payments repo
- [ ] Add `--service` flag to export commands
  - [ ] Export subgraph for a specific service
  - [ ] Useful for service-specific documentation

---

## Legend

- [x] Implemented
- [ ] Not started
- ✅ Complete / Better than graphify
- 🎯 **HIGH** priority
- 🔶 Medium priority (Phase 5-6)
- ⬜ Low priority / Future (Phase 7)
