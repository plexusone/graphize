// Package extract provides Go AST extraction for building knowledge graphs.
package extract

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/pkg/cache"
)

// Extractor extracts graph nodes and edges from Go source code.
type Extractor struct {
	fset  *token.FileSet
	cache *cache.Cache
}

// ExtractStats tracks extraction statistics.
type ExtractStats struct {
	TotalFiles  int
	CacheHits   int
	CacheMisses int
	Errors      int
}

// NewExtractor creates a new Go AST extractor.
func NewExtractor() *Extractor {
	return &Extractor{
		fset: token.NewFileSet(),
	}
}

// WithCache sets the cache for the extractor.
func (e *Extractor) WithCache(c *cache.Cache) *Extractor {
	e.cache = c
	return e
}

// ExtractDir extracts nodes and edges from all Go files in a directory tree.
func (e *Extractor) ExtractDir(dir string) (*graph.Graph, error) {
	g, _ := e.ExtractDirWithStats(dir)
	return g, nil
}

// ExtractDirWithStats extracts nodes and edges with cache statistics.
func (e *Extractor) ExtractDirWithStats(dir string) (*graph.Graph, *ExtractStats) {
	g := graph.NewGraph()
	stats := &ExtractStats{}

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and vendor
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files, skip tests for now
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		stats.TotalFiles++

		// Get relative path for cache key
		relPath, _ := filepath.Rel(dir, path)
		if relPath == "" {
			relPath = path
		}

		// Check cache first
		if e.cache != nil {
			if cached, ok := e.cache.Get(path, relPath); ok {
				stats.CacheHits++
				// Add cached nodes and edges to graph
				for _, n := range cached.Nodes {
					g.AddNode(n)
				}
				for _, edge := range cached.Edges {
					g.AddEdge(edge)
				}
				return nil
			}
			stats.CacheMisses++
		}

		// Extract from file
		nodes, edges, err := e.extractFileWithResults(path, dir, g)
		if err != nil {
			stats.Errors++
			return nil
		}

		// Save to cache
		if e.cache != nil && len(nodes) > 0 {
			_ = e.cache.Set(path, relPath, nodes, edges)
		}

		return nil
	})

	return g, stats
}

// extractFileWithResults extracts nodes and edges from a single Go file,
// returning the extracted items for caching.
func (e *Extractor) extractFileWithResults(path, baseDir string, g *graph.Graph) ([]*graph.Node, []*graph.Edge, error) {
	f, err := parser.ParseFile(e.fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	// Track nodes and edges for this file
	var nodes []*graph.Node
	var edges []*graph.Edge

	addNode := func(n *graph.Node) {
		nodes = append(nodes, n)
		g.AddNode(n)
	}

	addEdge := func(edge *graph.Edge) {
		edges = append(edges, edge)
		g.AddEdge(edge)
	}

	// Get relative path for cleaner IDs
	relPath, _ := filepath.Rel(baseDir, path)
	if relPath == "" {
		relPath = path
	}

	// Extract package node
	pkgName := f.Name.Name
	pkgID := makeID("pkg", pkgName)
	addNode(&graph.Node{
		ID:    pkgID,
		Type:  graph.NodeTypePackage,
		Label: pkgName,
		Attrs: map[string]string{
			"source_file": relPath,
		},
	})

	// Extract file node
	fileID := makeID("file", relPath)
	addNode(&graph.Node{
		ID:    fileID,
		Type:  graph.NodeTypeFile,
		Label: filepath.Base(path),
		Attrs: map[string]string{
			"path":    relPath,
			"package": pkgName,
		},
	})

	// File belongs to package
	addEdge(&graph.Edge{
		From:       fileID,
		To:         pkgID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract imports
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		importID := makeID("pkg", importPath)

		// Add import as node (external package)
		addNode(&graph.Node{
			ID:    importID,
			Type:  graph.NodeTypePackage,
			Label: filepath.Base(importPath),
			Attrs: map[string]string{
				"import_path": importPath,
				"external":    "true",
			},
		})

		// File imports package
		addEdge(&graph.Edge{
			From:       fileID,
			To:         importID,
			Type:       graph.EdgeTypeImports,
			Confidence: graph.ConfidenceExtracted,
			Attrs: map[string]string{
				"source_location": e.position(imp.Pos()),
			},
		})
	}

	// Extract declarations
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			e.extractFuncWithResults(d, relPath, pkgID, fileID, addNode, addEdge)
		case *ast.GenDecl:
			e.extractGenDeclWithResults(d, relPath, pkgID, fileID, addNode, addEdge)
		}
	}

	return nodes, edges, nil
}

// extractFuncWithResults extracts a function or method declaration with callbacks.
func (e *Extractor) extractFuncWithResults(fn *ast.FuncDecl, file, pkgID, fileID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	name := fn.Name.Name
	nodeType := graph.NodeTypeFunction

	// Check if it's a method (has receiver)
	var receiver string
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		nodeType = graph.NodeTypeMethod
		receiver = e.typeString(fn.Recv.List[0].Type)
	}

	// Create unique ID
	var funcID string
	if receiver != "" {
		funcID = makeID("method", receiver+"."+name)
	} else {
		funcID = makeID("func", file+"."+name)
	}

	// Add function node
	attrs := map[string]string{
		"source_file":     file,
		"source_location": e.position(fn.Pos()),
		"package":         pkgID,
	}
	if receiver != "" {
		attrs["receiver"] = receiver
	}
	if fn.Doc != nil {
		attrs["doc"] = fn.Doc.Text()
	}

	addNode(&graph.Node{
		ID:    funcID,
		Type:  nodeType,
		Label: name,
		Attrs: attrs,
	})

	// Function defined in file
	addEdge(&graph.Edge{
		From:       fileID,
		To:         funcID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// If method, link to receiver type
	if receiver != "" {
		typeID := makeID("type", receiver)
		addEdge(&graph.Edge{
			From:       funcID,
			To:         typeID,
			Type:       "method_of",
			Confidence: graph.ConfidenceExtracted,
		})
	}

	// Extract function calls within the body
	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				e.extractCallWithResults(call, funcID, file, addEdge)
			}
			return true
		})
	}
}

// extractCallWithResults extracts a function call with callback.
func (e *Extractor) extractCallWithResults(call *ast.CallExpr, callerID, file string, addEdge func(*graph.Edge)) {
	var calleeName string
	var calleeID string

	switch fn := call.Fun.(type) {
	case *ast.Ident:
		// Simple function call: foo()
		calleeName = fn.Name
		calleeID = makeID("func", file+"."+calleeName)
	case *ast.SelectorExpr:
		// Method or package call: obj.Method() or pkg.Func()
		if x, ok := fn.X.(*ast.Ident); ok {
			calleeName = x.Name + "." + fn.Sel.Name
			// Could be method call or package.Func
			calleeID = makeID("call", calleeName)
		}
	default:
		return
	}

	if calleeID == "" {
		return
	}

	// Add edge for the call
	addEdge(&graph.Edge{
		From:       callerID,
		To:         calleeID,
		Type:       graph.EdgeTypeCalls,
		Confidence: graph.ConfidenceExtracted,
		Attrs: map[string]string{
			"source_location": e.position(call.Pos()),
			"callee_name":     calleeName,
		},
	})
}

// extractGenDeclWithResults extracts type, const, and var declarations with callbacks.
func (e *Extractor) extractGenDeclWithResults(decl *ast.GenDecl, file, pkgID, fileID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			e.extractTypeWithResults(s, decl, file, pkgID, fileID, addNode, addEdge)
		case *ast.ValueSpec:
			e.extractValueWithResults(s, decl, file, pkgID, fileID, addNode, addEdge)
		}
	}
}

// extractTypeWithResults extracts a type declaration with callbacks.
func (e *Extractor) extractTypeWithResults(spec *ast.TypeSpec, decl *ast.GenDecl, file, pkgID, fileID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	name := spec.Name.Name
	typeID := makeID("type", name)

	// Determine specific type
	nodeType := "type"
	switch spec.Type.(type) {
	case *ast.StructType:
		nodeType = graph.NodeTypeStruct
	case *ast.InterfaceType:
		nodeType = graph.NodeTypeInterface
	}

	attrs := map[string]string{
		"source_file":     file,
		"source_location": e.position(spec.Pos()),
		"package":         pkgID,
	}
	if decl.Doc != nil {
		attrs["doc"] = decl.Doc.Text()
	}

	addNode(&graph.Node{
		ID:    typeID,
		Type:  nodeType,
		Label: name,
		Attrs: attrs,
	})

	// Type defined in file
	addEdge(&graph.Edge{
		From:       fileID,
		To:         typeID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract struct fields and their type references
	if st, ok := spec.Type.(*ast.StructType); ok {
		e.extractStructFieldsWithResults(st, typeID, addEdge)
	}

	// Extract interface methods
	if it, ok := spec.Type.(*ast.InterfaceType); ok {
		e.extractInterfaceMethodsWithResults(it, typeID, addNode, addEdge)
	}
}

// extractStructFieldsWithResults extracts field type references with callback.
func (e *Extractor) extractStructFieldsWithResults(st *ast.StructType, structID string, addEdge func(*graph.Edge)) {
	if st.Fields == nil {
		return
	}

	for _, field := range st.Fields.List {
		typeName := e.typeString(field.Type)
		if typeName == "" || isBuiltinType(typeName) {
			continue
		}

		typeID := makeID("type", typeName)
		addEdge(&graph.Edge{
			From:       structID,
			To:         typeID,
			Type:       graph.EdgeTypeReferences,
			Confidence: graph.ConfidenceExtracted,
			Attrs: map[string]string{
				"kind": "field_type",
			},
		})
	}
}

// extractInterfaceMethodsWithResults extracts method signatures with callbacks.
func (e *Extractor) extractInterfaceMethodsWithResults(it *ast.InterfaceType, ifaceID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	if it.Methods == nil {
		return
	}

	for _, method := range it.Methods.List {
		if len(method.Names) == 0 {
			// Embedded interface
			typeName := e.typeString(method.Type)
			if typeName != "" {
				embeddedID := makeID("type", typeName)
				addEdge(&graph.Edge{
					From:       ifaceID,
					To:         embeddedID,
					Type:       graph.EdgeTypeExtends,
					Confidence: graph.ConfidenceExtracted,
				})
			}
			continue
		}

		// Interface method
		for _, name := range method.Names {
			methodID := makeID("method", ifaceID+"."+name.Name)
			addNode(&graph.Node{
				ID:    methodID,
				Type:  graph.NodeTypeMethod,
				Label: name.Name,
				Attrs: map[string]string{
					"interface":       ifaceID,
					"source_location": e.position(method.Pos()),
				},
			})
			addEdge(&graph.Edge{
				From:       ifaceID,
				To:         methodID,
				Type:       graph.EdgeTypeContains,
				Confidence: graph.ConfidenceExtracted,
			})
		}
	}
}

// extractValueWithResults extracts const or var declarations with callbacks.
func (e *Extractor) extractValueWithResults(spec *ast.ValueSpec, decl *ast.GenDecl, file, pkgID, fileID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	nodeType := graph.NodeTypeVariable
	if decl.Tok == token.CONST {
		nodeType = graph.NodeTypeConstant
	}

	for _, name := range spec.Names {
		if name.Name == "_" {
			continue // Skip blank identifier
		}

		valueID := makeID(nodeType, file+"."+name.Name)
		addNode(&graph.Node{
			ID:    valueID,
			Type:  nodeType,
			Label: name.Name,
			Attrs: map[string]string{
				"source_file":     file,
				"source_location": e.position(name.Pos()),
				"package":         pkgID,
			},
		})

		addEdge(&graph.Edge{
			From:       fileID,
			To:         valueID,
			Type:       graph.EdgeTypeContains,
			Confidence: graph.ConfidenceExtracted,
		})
	}
}

// typeString returns a string representation of a type expression.
func (e *Extractor) typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return e.typeString(t.X)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	case *ast.ArrayType:
		return e.typeString(t.Elt)
	case *ast.MapType:
		return "map"
	case *ast.ChanType:
		return "chan"
	case *ast.FuncType:
		return "func"
	}
	return ""
}

// position returns a string representation of a token position.
func (e *Extractor) position(pos token.Pos) string {
	p := e.fset.Position(pos)
	return p.String()
}

// makeID creates a stable, filesystem-safe ID.
func makeID(prefix, name string) string {
	// Replace problematic characters
	safe := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', ' ':
			return '_'
		case '.':
			return '.'
		default:
			return r
		}
	}, name)
	return prefix + "_" + safe
}

// isBuiltinType returns true for Go builtin types.
func isBuiltinType(name string) bool {
	builtins := map[string]bool{
		"bool":       true,
		"byte":       true,
		"complex64":  true,
		"complex128": true,
		"error":      true,
		"float32":    true,
		"float64":    true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"rune":       true,
		"string":     true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
		"any":        true,
	}
	return builtins[name]
}
