# Go Analyzer Reference

The Go analyzer is Graphize's reference implementation, providing full semantic analysis using Go's native toolchain.

## Overview

The Go analyzer exists in two variants:

| Variant | Package | Parser | Semantic Depth |
|---------|---------|--------|----------------|
| Basic | `extractor.go` | `go/ast` | Structural only |
| Semantic | `semantic.go` | `go/ast` + `go/types` | Full type resolution |

## Basic Extractor

The basic extractor uses `go/ast` for fast, dependency-free parsing:

```go
import (
    "go/ast"
    "go/parser"
    "go/token"
)

func (e *Extractor) ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error) {
    fset := token.NewFileSet()
    f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
    // ...
}
```

### Extracted Elements

| Element | Node Type | Description |
|---------|-----------|-------------|
| Packages | `package` | Go packages |
| Files | `file` | Source files |
| Functions | `function` | Top-level functions |
| Methods | `method` | Type methods |
| Structs | `struct` | Struct types |
| Interfaces | `interface` | Interface types |
| Constants | `constant` | Const declarations |
| Variables | `variable` | Var declarations |

### Extracted Edges

| Edge Type | Description |
|-----------|-------------|
| `contains` | File contains declaration |
| `imports` | File imports package |
| `calls` | Function calls function |
| `method_of` | Method belongs to type |
| `references` | Struct field references type |
| `extends` | Interface embeds interface |

### Limitations

- Call edges are string-based (no type resolution)
- Cannot determine which package a selector refers to
- No interface implementation detection
- Framework detection not available

## Semantic Extractor

The semantic extractor adds `go/types` for full type checking:

```go
import (
    "go/types"
    "golang.org/x/tools/go/packages"
)

func (e *SemanticExtractor) LoadPackage(dir string) error {
    cfg := &packages.Config{
        Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
        Dir:  dir,
    }
    pkgs, err := packages.Load(cfg, "./...")
    // ...
}
```

### Additional Capabilities

| Feature | Description |
|---------|-------------|
| Type-resolved calls | Know exactly which `pkg.Func` is called |
| Interface detection | Find which interfaces a type implements |
| Method resolution | Resolve method calls to receiver types |
| Framework detection | Detect Gin, Echo, Chi, gRPC, etc. |
| Handler detection | Mark HTTP handlers as entry points |

### Type-Resolved Call Edges

Basic extractor:
```
funcA → go_call_http.Get  (string-based, may be wrong)
```

Semantic extractor:
```
funcA → go_func_net/http.Get  (fully qualified, correct)
```

### Interface Implementation

The semantic extractor detects when types implement interfaces:

```go
type Reader interface {
    Read([]byte) (int, error)
}

type MyReader struct{}

func (r *MyReader) Read(p []byte) (int, error) { ... }
```

Produces edge:
```
go_type_MyReader → go_type_io.Reader  (implements)
```

### Framework Detection

Detects common Go frameworks from imports:

| Framework | Import Path | Handler Type |
|-----------|-------------|--------------|
| Gin | `github.com/gin-gonic/gin` | `gin.HandlerFunc` |
| Echo | `github.com/labstack/echo` | `echo.HandlerFunc` |
| Fiber | `github.com/gofiber/fiber` | `fiber.Handler` |
| Chi | `github.com/go-chi/chi` | `http.HandlerFunc` |
| gRPC | `google.golang.org/grpc` | Service methods |

### Handler Detection

HTTP handlers are marked as entry points:

```go
func (e *SemanticExtractor) detectHandlerType(sig *types.Signature) string {
    // Check for http.HandlerFunc pattern
    // Check for gin.Context pattern
    // etc.
}
```

Node attributes for handlers:
```json
{
    "handler_type": "gin.HandlerFunc",
    "is_entrypoint": "true"
}
```

## Usage

### Basic Extraction (Default)

```bash
graphize analyze
```

### Semantic Extraction

```go
// In your code
extractor := golang.NewSemanticExtractor()
err := extractor.LoadPackage("./")
nodes, edges, err := extractor.ExtractFile(path, baseDir)
```

### Programmatic Access

```go
import (
    "github.com/plexusone/graphize/pkg/extract/golang"
)

func main() {
    // Create semantic extractor
    e := golang.NewSemanticExtractor(
        golang.WithFrameworkDetection(true),
    )

    // Load package with type information
    if err := e.LoadPackage("./myproject"); err != nil {
        log.Fatal(err)
    }

    // Extract individual files
    nodes, edges, err := e.ExtractFile("main.go", "./myproject")

    // Or use with MultiExtractor
    multi := extract.NewMultiExtractor("./myproject", nil)
    // The semantic extractor will be used for .go files
}
```

## Node ID Format

Go nodes use the `go_` prefix:

| Element | ID Format | Example |
|---------|-----------|---------|
| Package | `go_pkg_{import_path}` | `go_pkg_net/http` |
| File | `go_file_{rel_path}` | `go_file_pkg/handlers/user.go` |
| Function | `go_func_{pkg}.{name}` | `go_func_main.go.main` |
| Method | `go_method_{type}.{name}` | `go_method_UserService.Create` |
| Type | `go_type_{pkg}.{name}` | `go_type_User` |

## Attributes

### Function/Method Attributes

```json
{
    "source_file": "pkg/handlers/user.go",
    "source_location": "user.go:42:1",
    "package": "go_pkg_handlers",
    "language": "go",
    "signature": "func(ctx context.Context, id string) (*User, error)",
    "receiver": "UserService",
    "exported": "true",
    "doc": "GetUser retrieves a user by ID.",
    "handler_type": "gin.HandlerFunc",
    "is_entrypoint": "true"
}
```

### Package Attributes

```json
{
    "import_path": "github.com/myorg/myapp/handlers",
    "language": "go",
    "stdlib": "false",
    "external": "true"
}
```

## Future Enhancements

### SSA Analysis

Adding `golang.org/x/tools/go/ssa` for:

- Control flow graphs
- Data flow analysis
- Precise call graphs (handling interfaces)
- Dead code detection

### Taint Analysis

Tracking data flow for security:

- User input sources
- Dangerous sinks (SQL, exec, etc.)
- Sanitization points

### Build Tag Support

Extracting from different build configurations:

```go
cfg := &packages.Config{
    BuildFlags: []string{"-tags=integration"},
}
```
