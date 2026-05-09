# Creating Analyzers

This guide walks through creating a new language analyzer for Graphize.

## Quick Start

Create a new Go module for your analyzer:

```bash
mkdir graphize-mylang
cd graphize-mylang
go mod init github.com/yourorg/graphize-mylang
go get github.com/plexusone/graphize/provider
go get github.com/plexusone/graphfs/pkg/graph
```

## Project Structure

```
graphize-mylang/
├── go.mod
├── go.sum
├── extractor.go      # Main extractor implementation
├── extractor_test.go # Tests
├── parser.go         # Language-specific parsing (optional)
└── README.md
```

## Implementing the Interface

### Step 1: Define the Extractor

```go
package mylang

import (
    "path/filepath"
    "strings"

    "github.com/plexusone/graphfs/pkg/graph"
    "github.com/plexusone/graphize/provider"
)

const (
    Language   = "mylang"
    NodePrefix = "mylang_"
)

// Extractor implements provider.LanguageExtractor for MyLang.
type Extractor struct {
    // Add any state needed for extraction
}

// New creates a new MyLang extractor.
func New() *Extractor {
    return &Extractor{}
}
```

### Step 2: Implement Required Methods

```go
// Language returns the canonical language name.
func (e *Extractor) Language() string {
    return Language
}

// Extensions returns file extensions this extractor handles.
func (e *Extractor) Extensions() []string {
    return []string{".ml", ".mli"}
}

// CanExtract returns true for MyLang files.
func (e *Extractor) CanExtract(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    return ext == ".ml" || ext == ".mli"
}
```

### Step 3: Implement ExtractFile

This is the core method that extracts nodes and edges:

```go
func (e *Extractor) ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error) {
    // Parse the file
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, nil, err
    }

    var nodes []*graph.Node
    var edges []*graph.Edge

    // Helper functions
    addNode := func(n *graph.Node) { nodes = append(nodes, n) }
    addEdge := func(e *graph.Edge) { edges = append(edges, e) }

    // Get relative path for stable IDs
    relPath, _ := filepath.Rel(baseDir, path)
    if relPath == "" {
        relPath = path
    }

    // Create file node
    fileID := makeID("file", relPath)
    addNode(&graph.Node{
        ID:    fileID,
        Type:  graph.NodeTypeFile,
        Label: filepath.Base(path),
        Attrs: map[string]string{
            "path":     relPath,
            "language": Language,
        },
    })

    // Parse and extract declarations
    // ... your parsing logic here ...

    return nodes, edges, nil
}

// makeID creates a stable, prefixed node ID.
func makeID(nodeType, name string) string {
    safe := strings.Map(func(r rune) rune {
        switch r {
        case '/', '\\', ':', '*', '?', '"', '<', '>', '|', ' ':
            return '_'
        default:
            return r
        }
    }, name)
    return NodePrefix + nodeType + "_" + safe
}
```

### Step 4: Implement Framework Detection (Optional)

```go
func (e *Extractor) DetectFramework(path string) *provider.FrameworkInfo {
    // Return nil if no framework detection
    return nil
}
```

### Step 5: Register the Extractor

Use an `init()` function to register with the global registry:

```go
func init() {
    provider.Register(func() provider.LanguageExtractor {
        return New()
    }, provider.PriorityDefault)
}
```

## Node Types

Use standard node types from `graphfs/pkg/graph`:

| Constant | Value | Description |
|----------|-------|-------------|
| `NodeTypePackage` | `"package"` | Module/package |
| `NodeTypeFile` | `"file"` | Source file |
| `NodeTypeFunction` | `"function"` | Function/procedure |
| `NodeTypeMethod` | `"method"` | Method on a type |
| `NodeTypeStruct` | `"struct"` | Struct/class |
| `NodeTypeInterface` | `"interface"` | Interface/protocol |
| `NodeTypeVariable` | `"variable"` | Variable |
| `NodeTypeConstant` | `"constant"` | Constant |

## Edge Types

Standard edge types:

| Constant | Value | Description |
|----------|-------|-------------|
| `EdgeTypeContains` | `"contains"` | Parent contains child |
| `EdgeTypeImports` | `"imports"` | File imports package |
| `EdgeTypeCalls` | `"calls"` | Function calls function |
| `EdgeTypeReferences` | `"references"` | Type references type |
| `EdgeTypeImplements` | `"implements"` | Type implements interface |
| `EdgeTypeExtends` | `"extends"` | Type extends type |

## Confidence Levels

Mark edges with confidence based on extraction method:

| Constant | Description |
|----------|-------------|
| `ConfidenceExtracted` | Deterministically extracted from AST |
| `ConfidenceInferred` | Inferred (LLM, heuristics) |
| `ConfidenceAmbiguous` | Low confidence inference |

## Using Your Analyzer

### In graphize CLI

Add a blank import to `cmd/graphize/cmd/analyze.go`:

```go
import (
    // Built-in extractors
    _ "github.com/plexusone/graphize/pkg/extract/golang"
    _ "github.com/plexusone/graphize/pkg/extract/java"

    // External extractors
    _ "github.com/yourorg/graphize-mylang"
)
```

### In Custom Tools

```go
package main

import (
    "github.com/plexusone/graphize/pkg/extract"
    _ "github.com/yourorg/graphize-mylang" // Register on import
)

func main() {
    extractor := extract.NewMultiExtractor(baseDir, nil)
    nodes, edges, err := extractor.ExtractDir("./src")
    // ...
}
```

## Testing

Write tests to verify extraction:

```go
func TestExtractFunction(t *testing.T) {
    e := New()

    // Create test file
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.ml")
    os.WriteFile(testFile, []byte(`
        func hello() {
            print("Hello")
        }
    `), 0644)

    nodes, edges, err := e.ExtractFile(testFile, tmpDir)
    require.NoError(t, err)

    // Verify function node exists
    var funcNode *graph.Node
    for _, n := range nodes {
        if n.Type == graph.NodeTypeFunction && n.Label == "hello" {
            funcNode = n
            break
        }
    }
    require.NotNil(t, funcNode)
}
```

## Best Practices

### Node ID Stability

- Use relative paths, not absolute
- Prefix with language code (`go_`, `java_`, etc.)
- Make IDs deterministic (same input = same ID)

### Attributes

Include useful metadata:

```go
attrs := map[string]string{
    "source_file":     relPath,
    "source_location": "file.ml:42:5",
    "language":        Language,
    "exported":        "true",      // if public/exported
    "doc":             "...",        // documentation comment
}
```

### Error Handling

- Return partial results if possible
- Log warnings for non-fatal issues
- Return errors only for unrecoverable failures

### Performance

- Use caching for expensive operations
- Process files incrementally when possible
- Consider lazy loading for large codebases

## Next Steps

- [Semantic Analysis](semantic-analysis.md) - Add type information
- [Go Analyzer Reference](go-analyzer.md) - See a complete example
