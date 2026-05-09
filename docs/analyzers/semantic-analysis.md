# Semantic Analysis

This guide explains how to add semantic depth (type information, call resolution, interface detection) to language analyzers.

## Why Semantic Analysis?

| Level | What You Get | Use Cases |
|-------|--------------|-----------|
| **Structural** | Syntax tree, basic edges | Code navigation, simple call graphs |
| **Semantic** | Types, resolved calls | Security analysis, refactoring |
| **Data Flow** | Taint tracking, CFG | Vulnerability detection |

For security and reachability analysis, semantic depth is essential.

## Parser Comparison

Different parsing approaches offer different capabilities:

| Parser Type | Example | Types | Calls | Interfaces | Speed |
|-------------|---------|-------|-------|------------|-------|
| Regex/text | Markdown | No | No | No | Fast |
| Tree-sitter | Java, TS | No | Partial | No | Fast |
| Native AST | `go/ast` | No | Partial | No | Medium |
| Type Checker | `go/types` | Yes | Yes | Yes | Slow |
| Compiler API | TS Compiler | Yes | Yes | Yes | Slow |

## Adding Types to Your Analyzer

### Pattern 1: Separate Semantic Extractor

Create a separate extractor for semantic analysis:

```go
// extractor.go - fast, structural
type Extractor struct {
    fset *token.FileSet
}

// semantic.go - full semantic analysis
type SemanticExtractor struct {
    fset     *token.FileSet
    typeInfo map[string]*types.Info
}
```

### Pattern 2: Optional Semantic Mode

Use options to enable semantic analysis:

```go
type Extractor struct {
    fset           *token.FileSet
    semanticMode   bool
    typeInfo       *types.Info
}

func WithSemanticAnalysis(enable bool) Option {
    return func(e *Extractor) {
        e.semanticMode = enable
    }
}
```

## Go: Using go/types

### Loading Packages

```go
import "golang.org/x/tools/go/packages"

func (e *SemanticExtractor) LoadPackage(dir string) error {
    cfg := &packages.Config{
        Mode: packages.NeedName |
              packages.NeedFiles |
              packages.NeedImports |
              packages.NeedTypes |
              packages.NeedTypesInfo |
              packages.NeedSyntax,
        Dir: dir,
    }

    pkgs, err := packages.Load(cfg, "./...")
    if err != nil {
        return err
    }

    // Cache type information
    for _, pkg := range pkgs {
        e.pkgCache[pkg.PkgPath] = pkg
    }
    return nil
}
```

### Resolving Function Calls

```go
func (e *SemanticExtractor) resolveCall(call *ast.CallExpr, pkg *packages.Package) string {
    switch fn := call.Fun.(type) {
    case *ast.SelectorExpr:
        // Method call or qualified function
        if sel := pkg.TypesInfo.Selections[fn]; sel != nil {
            // Method call
            if f, ok := sel.Obj().(*types.Func); ok {
                return f.FullName() // e.g., "(*http.Client).Get"
            }
        }
        if obj := pkg.TypesInfo.Uses[fn.Sel]; obj != nil {
            // Qualified function call
            if f, ok := obj.(*types.Func); ok {
                return f.FullName() // e.g., "net/http.Get"
            }
        }
    case *ast.Ident:
        // Local function
        if obj := pkg.TypesInfo.Uses[fn]; obj != nil {
            if f, ok := obj.(*types.Func); ok {
                return f.FullName()
            }
        }
    }
    return ""
}
```

### Detecting Interface Implementations

```go
func (e *SemanticExtractor) findImplementations(t types.Type) []*types.Interface {
    var implemented []*types.Interface

    for _, pkg := range e.pkgCache {
        scope := pkg.Types.Scope()
        for _, name := range scope.Names() {
            obj := scope.Lookup(name)
            if tn, ok := obj.(*types.TypeName); ok {
                if iface, ok := tn.Type().Underlying().(*types.Interface); ok {
                    // Check if t implements iface
                    if types.Implements(t, iface) {
                        implemented = append(implemented, iface)
                    }
                    // Also check pointer type
                    if types.Implements(types.NewPointer(t), iface) {
                        implemented = append(implemented, iface)
                    }
                }
            }
        }
    }
    return implemented
}
```

## TypeScript: Using the Compiler API

For TypeScript, use the TypeScript compiler API via a subprocess or WASM:

### Subprocess Approach

```go
func (e *TSExtractor) getTypeInfo(file string) (*TypeInfo, error) {
    // Run ts-analyzer tool
    cmd := exec.Command("npx", "ts-analyzer", file, "--json")
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    var info TypeInfo
    if err := json.Unmarshal(output, &info); err != nil {
        return nil, err
    }
    return &info, nil
}
```

### TypeScript Analyzer Tool (Node.js)

```typescript
// ts-analyzer.ts
import * as ts from 'typescript';

function analyzeFile(filePath: string) {
    const program = ts.createProgram([filePath], {});
    const checker = program.getTypeChecker();
    const sourceFile = program.getSourceFile(filePath);

    const symbols: SymbolInfo[] = [];

    ts.forEachChild(sourceFile, function visit(node) {
        if (ts.isFunctionDeclaration(node) && node.name) {
            const symbol = checker.getSymbolAtLocation(node.name);
            const type = checker.getTypeOfSymbolAtLocation(symbol, node);
            symbols.push({
                name: node.name.text,
                type: checker.typeToString(type),
                kind: 'function'
            });
        }
        ts.forEachChild(node, visit);
    });

    console.log(JSON.stringify(symbols));
}
```

## Java: Using Tree-sitter + Symbol Resolution

For Java, combine Tree-sitter with import resolution:

```go
func (e *JavaExtractor) resolveType(typeName string, imports map[string]string) string {
    // Check explicit imports
    if fullPath, ok := imports[typeName]; ok {
        return fullPath
    }

    // Check wildcard imports
    for pkg, path := range imports {
        if strings.HasSuffix(path, ".*") {
            // Could be from this package
            return strings.TrimSuffix(path, "*") + typeName
        }
    }

    // java.lang types are auto-imported
    if isJavaLangType(typeName) {
        return "java.lang." + typeName
    }

    return typeName
}
```

## Framework Detection

### Pattern Matching

Detect frameworks by analyzing imports and patterns:

```go
type FrameworkDetector struct {
    patterns map[string]FrameworkPattern
}

type FrameworkPattern struct {
    ImportPrefix  string
    Annotations   []string
    Conventions   []string
    FrameworkName string
}

var goFrameworks = []FrameworkPattern{
    {
        ImportPrefix:  "github.com/gin-gonic/gin",
        FrameworkName: "Gin",
    },
    {
        ImportPrefix:  "github.com/labstack/echo",
        FrameworkName: "Echo",
    },
}

var javaFrameworks = []FrameworkPattern{
    {
        ImportPrefix:  "org.springframework",
        Annotations:   []string{"@Controller", "@Service", "@Repository"},
        FrameworkName: "Spring",
    },
}
```

### Handler/Entry Point Detection

Mark HTTP handlers and entry points:

```go
func detectGoHandler(sig *types.Signature) string {
    params := sig.Params()

    // http.HandlerFunc: func(http.ResponseWriter, *http.Request)
    if params.Len() == 2 {
        p0 := params.At(0).Type().String()
        p1 := params.At(1).Type().String()
        if strings.Contains(p0, "ResponseWriter") &&
           strings.Contains(p1, "Request") {
            return "http.HandlerFunc"
        }
    }

    // gin.HandlerFunc: func(*gin.Context)
    if params.Len() == 1 {
        p0 := params.At(0).Type().String()
        if strings.Contains(p0, "gin.Context") {
            return "gin.HandlerFunc"
        }
    }

    return ""
}
```

## Performance Considerations

### Lazy Loading

Don't load type info until needed:

```go
func (e *SemanticExtractor) ensureLoaded(pkg string) error {
    if _, ok := e.pkgCache[pkg]; ok {
        return nil
    }
    return e.loadPackage(pkg)
}
```

### Caching

Cache expensive computations:

```go
type SemanticExtractor struct {
    // Cache interface implementations
    implCache map[string][]string // type -> []interface

    // Cache resolved calls
    callCache map[string]string // call site -> target
}
```

### Parallel Processing

Process packages in parallel:

```go
func (e *SemanticExtractor) LoadPackages(dirs []string) error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(dirs))

    for _, dir := range dirs {
        wg.Add(1)
        go func(d string) {
            defer wg.Done()
            if err := e.loadPackage(d); err != nil {
                errCh <- err
            }
        }(dir)
    }

    wg.Wait()
    close(errCh)

    for err := range errCh {
        return err // Return first error
    }
    return nil
}
```

## Best Practices

1. **Graceful Degradation**: Fall back to structural extraction if semantic analysis fails
2. **Incremental Updates**: Only re-analyze changed files
3. **Memory Management**: Clear caches for large codebases
4. **Error Handling**: Log but don't fail on partial analysis errors
5. **Confidence Levels**: Mark semantically-derived edges as `EXTRACTED` (high confidence)
