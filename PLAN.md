# Graphize Implementation Plan

## Current Status

**Version:** v0.1.0 (released 2026-04-11)

Phases 1-4 are complete:
- ✅ Phase 1: MVP (source tracking, AST extraction, query, export)
- ✅ Phase 2: LLM Semantic Extraction (caching, enhance, merge)
- ✅ Phase 3: Analysis & Reports (community detection, god nodes, surprises)
- ✅ Phase 4: Agent Integration (MCP server, AGENTS folder)

## Implementation Strategy

**Decision: Features before Languages**

Quick wins in Phase 5 benefit all users (including future TypeScript/Swift users).
Multi-language support requires more architectural work and comes after.

```
┌─────────────────────────────────────────────────────────────────┐
│                    IMPLEMENTATION ORDER                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Phase 5: Quick Wins (~1-2 days each)                           │
│  ├── graphize path "A" "B"      ← Use existing FindPath         │
│  ├── graphize benchmark         ← Token counting                │
│  ├── --directed flag            ← Minor graph change            │
│  ├── Git hooks                  ← Shell script generation       │
│  ├── Watch mode                 ← fsnotify integration          │
│  └── Obsidian export            ← Markdown file generation      │
│                                                                  │
│  Phase 6: Enhanced Analysis                                      │
│  ├── Betweenness centrality     ← gonum/graph                   │
│  ├── graphize explain           ← Node context summary          │
│  └── Platform installers        ← codex, cursor, gemini, etc.   │
│                                                                  │
│  Phase 7: Multi-language                                         │
│  ├── go-tree-sitter setup       ← CGo bindings                  │
│  ├── TypeScript extractor       ← High demand                   │
│  ├── Swift extractor            ← iOS/macOS ecosystem           │
│  └── Additional languages       ← Python, Rust, Java            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 5: Quick Wins

### 5.1 Path Command

**Goal:** Trace exact path between two nodes.

```bash
graphize path "func_main" "pkg_utils"
```

**Implementation:**
```go
// cmd/graphize/cmd/path.go

var pathCmd = &cobra.Command{
    Use:   "path <from> <to>",
    Short: "Find shortest path between two nodes",
    RunE:  runPath,
}

func runPath(cmd *cobra.Command, args []string) error {
    // Load graph
    // Use query.NewTraverser(g).FindPath(from, to)
    // Display path with edge types
}
```

**Effort:** 1-2 hours

---

### 5.2 Benchmark Command

**Goal:** Show token reduction statistics.

```bash
graphize benchmark
# Output:
# Raw corpus:     1,234,567 tokens
# TOON export:       12,345 tokens
# Reduction:            99x
```

**Implementation:**
```go
// cmd/graphize/cmd/benchmark.go

func runBenchmark(cmd *cobra.Command, args []string) error {
    // Count tokens in source files (rough: words * 1.3)
    // Count tokens in TOON export
    // Display reduction ratio
}
```

**Effort:** 1-2 hours

---

### 5.3 Directed Graphs

**Goal:** Preserve edge direction for call graph analysis.

```bash
graphize analyze --directed
```

**Implementation:**
- Add `--directed` flag to analyze command
- Store direction metadata in graph
- Affects traversal (outgoing vs incoming matters more)

**Effort:** 2-3 hours

---

### 5.4 Git Hooks

**Goal:** Auto-analyze on commit, check staleness on checkout.

```bash
graphize hook install    # Install post-commit and post-checkout hooks
graphize hook uninstall  # Remove hooks
graphize hook status     # Check installation
```

**Implementation:**
```go
// cmd/graphize/cmd/hook.go

func installHooks() error {
    // Write .git/hooks/post-commit
    // Write .git/hooks/post-checkout
    // Make executable
}
```

**post-commit hook:**
```bash
#!/bin/bash
graphize analyze --quiet
```

**post-checkout hook:**
```bash
#!/bin/bash
graphize status --check || echo "Graph may be stale. Run: graphize analyze"
```

**Effort:** 2-3 hours

---

### 5.5 Watch Mode

**Goal:** Auto-rebuild on file changes.

```bash
graphize watch           # Watch and rebuild
graphize watch --html    # Also regenerate HTML
```

**Implementation:**
```go
// cmd/graphize/cmd/watch.go

import "github.com/fsnotify/fsnotify"

func runWatch(cmd *cobra.Command, args []string) error {
    watcher, _ := fsnotify.NewWatcher()
    // Add source directories
    // Debounce events (500ms)
    // Run analyze on change
}
```

**Effort:** 3-4 hours

---

### 5.6 Obsidian Export

**Goal:** Generate wiki-style vault for Obsidian.

```bash
graphize export obsidian -o ./vault
```

**Output structure:**
```
vault/
├── index.md              # Entry point with god nodes
├── communities/
│   ├── community-1.md    # Community overview + members
│   └── community-2.md
└── nodes/
    ├── func_main.md      # Node details with wikilinks
    └── pkg_utils.md
```

**Implementation:**
```go
// cmd/graphize/cmd/export_obsidian.go

func exportObsidian(nodes, edges, communities, outputDir) error {
    // Generate index.md with [[wikilinks]]
    // Generate community pages
    // Generate node pages with neighbors
}
```

**Effort:** 3-4 hours

---

### 5.7 Neo4j Export

**Goal:** Generate Cypher statements for Neo4j import.

```bash
graphize export cypher -o graph.cypher
```

**Output:**
```cypher
CREATE (n:Node {id: "func_main", type: "function", label: "main"});
CREATE (n:Node {id: "pkg_utils", type: "package", label: "utils"});
CREATE (a)-[:CALLS {confidence: "EXTRACTED"}]->(b)
  WHERE a.id = "func_main" AND b.id = "func_helper";
```

**Effort:** 2-3 hours

---

## Phase 6: Enhanced Analysis

### 6.1 Betweenness Centrality

Use gonum/graph to identify bridge nodes.

```go
import "gonum.org/v1/gonum/graph/network"

func BetweennessCentrality(g graph.Graph) map[string]float64 {
    // Convert to gonum graph
    // Calculate betweenness
    // Return node ID -> centrality score
}
```

### 6.2 Explain Command

```bash
graphize explain "func_main"
# Output:
# Node: func_main (function)
# Community: 3 (cli commands)
# In-degree: 2, Out-degree: 15
# Neighbors: func_init, func_run, pkg_cobra...
# Called by: main
# Calls: runAnalyze, runExport, runQuery...
```

### 6.3 Platform Installers

```bash
graphize install claude   # Already have MCP
graphize install codex    # hooks.json + AGENTS.md
graphize install cursor   # .cursor/rules/graphify.mdc
graphize install gemini   # .gemini/settings.json
graphize install copilot  # ~/.copilot/skills/graphize/
```

---

## Phase 7: Multi-language Support

### 7.1 Tree-sitter Setup

```go
import sitter "github.com/smacker/go-tree-sitter"

// Language grammars
import (
    "github.com/smacker/go-tree-sitter/typescript"
    "github.com/smacker/go-tree-sitter/swift"
)
```

### 7.2 Language Extractors

```go
// pkg/extract/typescript.go
func ExtractTypeScript(path string) (*Extraction, error) {
    parser := sitter.NewParser()
    parser.SetLanguage(typescript.GetLanguage())
    // Parse and extract nodes/edges
}

// pkg/extract/swift.go
func ExtractSwift(path string) (*Extraction, error) {
    parser := sitter.NewParser()
    parser.SetLanguage(swift.GetLanguage())
    // Parse and extract nodes/edges
}
```

### 7.3 Unified Node ID Scheme

```
go:func_main           # Go function
ts:class_UserService   # TypeScript class
swift:struct_User      # Swift struct
```

---

## Implementation Schedule

### Week 1: Phase 5 Quick Wins

| Day | Task | Effort |
|-----|------|--------|
| 1 | `graphize path` command | 2h |
| 1 | `graphize benchmark` command | 2h |
| 2 | `--directed` flag | 3h |
| 2 | `graphize hook` commands | 3h |
| 3 | `graphize watch` mode | 4h |
| 4 | `graphize export obsidian` | 4h |
| 4 | `graphize export cypher` | 3h |
| 5 | Testing and refinement | 4h |

### Week 2: Phase 6 Enhanced Analysis

| Day | Task | Effort |
|-----|------|--------|
| 1 | Betweenness centrality | 3h |
| 2 | `graphize explain` command | 3h |
| 3-4 | Platform installers | 6h |
| 5 | Testing and documentation | 4h |

### Week 3+: Phase 7 Multi-language

| Task | Effort |
|------|--------|
| go-tree-sitter setup | 4h |
| TypeScript extractor | 6h |
| Swift extractor | 6h |
| Cross-language testing | 4h |

---

## Success Criteria

### Phase 5
- [ ] `graphize path A B` shows shortest path
- [ ] `graphize benchmark` shows token reduction
- [ ] `--directed` preserves edge direction
- [ ] Git hooks auto-analyze on commit
- [ ] Watch mode rebuilds on file change
- [ ] Obsidian vault has working wikilinks
- [ ] Neo4j Cypher imports successfully

### Phase 6
- [x] Betweenness identifies bridge nodes
- [ ] Explain shows useful node context
- [ ] Platform installers work for top 3 platforms

### Phase 7
- [ ] TypeScript extraction matches Go quality
- [ ] Swift extraction works for iOS projects
- [ ] Mixed-language repos produce unified graph

---

## Next Steps

1. Start with `graphize path` (uses existing FindPath)
2. Add `graphize benchmark` (simple token counting)
3. Continue through Phase 5 in order
