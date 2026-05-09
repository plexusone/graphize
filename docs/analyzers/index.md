# Language Analyzers

Graphize uses a pluggable analyzer architecture to support multiple programming languages. Each analyzer extracts nodes and edges from source code to build a unified knowledge graph.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     ANALYZER ARCHITECTURE                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │ Go Analyzer  │    │ Java Analyzer│    │ TS Analyzer  │       │
│  │ (go/ast)     │    │ (tree-sitter)│    │ (tree-sitter)│       │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘       │
│         │                   │                   │                │
│         └───────────────────┼───────────────────┘                │
│                             ▼                                    │
│                   ┌──────────────────┐                          │
│                   │ Provider Registry│                          │
│                   │ (priority-based) │                          │
│                   └────────┬─────────┘                          │
│                            │                                     │
│                            ▼                                     │
│                   ┌──────────────────┐                          │
│                   │ MultiExtractor   │                          │
│                   │ (orchestrator)   │                          │
│                   └────────┬─────────┘                          │
│                            │                                     │
│                            ▼                                     │
│                   ┌──────────────────┐                          │
│                   │ GraphFS Store    │                          │
│                   │ (nodes + edges)  │                          │
│                   └──────────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
```

## Built-in Analyzers

| Language | Parser | Semantic Depth | Status |
|----------|--------|----------------|--------|
| **Go** | `go/ast` + `go/types` | Full (types, interfaces, SSA) | Production |
| **Java** | Tree-sitter | Structural | Production |
| **TypeScript** | Tree-sitter | Structural | Production |
| **Swift** | Tree-sitter | Structural | Production |
| **Markdown** | Regex/text | Documentation | Production |

## External Analyzers

External analyzers can be created as separate Go modules:

| Analyzer | Language | Status |
|----------|----------|--------|
| [graphize-groovy](https://github.com/plexusone/graphize-groovy) | Groovy/Grails | Planned |

## Provider Interface

All analyzers implement the `LanguageExtractor` interface:

```go
type LanguageExtractor interface {
    // Language returns the canonical language name
    Language() string

    // Extensions returns file extensions handled
    Extensions() []string

    // CanExtract returns true if this extractor handles the file
    CanExtract(path string) bool

    // ExtractFile extracts nodes and edges from a source file
    ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error)

    // DetectFramework returns detected framework info
    DetectFramework(path string) *FrameworkInfo
}
```

## Priority System

Analyzers register with a priority level. Higher priority analyzers override lower ones for the same file extension:

| Priority | Constant | Use Case |
|----------|----------|----------|
| 0 | `PriorityDefault` | Built-in analyzers |
| 10 | `PriorityThick` | SDK-based analyzers (richer extraction) |
| 100 | `PriorityCustom` | User-provided custom analyzers |

## Extraction Levels

### Level 1: Structural (Tree-sitter)

- Fast, incremental parsing
- Syntax tree only (no types)
- Good for: code navigation, basic call graphs

### Level 2: Semantic (go/types, TypeScript API)

- Type resolution
- Interface implementation detection
- Framework detection
- Good for: security analysis, reachability

### Level 3: Data Flow (SSA, taint analysis)

- Control flow graphs
- Data flow tracking
- Taint propagation
- Good for: vulnerability detection

## Next Steps

- [Creating Analyzers](creating-analyzers.md) - Build your own analyzer
- [Go Analyzer Reference](go-analyzer.md) - Deep dive into Go analysis
- [Semantic Analysis](semantic-analysis.md) - Adding type information
