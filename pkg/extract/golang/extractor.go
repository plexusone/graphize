// Package golang provides Go language extraction for knowledge graphs.
package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/provider"
)

const (
	// Language is the canonical name for Go.
	Language = "go"

	// NodePrefix is the prefix for all Go node IDs.
	NodePrefix = "go_"
)

// Extractor implements extract.LanguageExtractor for Go source code.
// It uses the native go/parser and go/ast packages for extraction.
type Extractor struct {
	fset *token.FileSet
}

// New creates a new Go extractor.
func New() *Extractor {
	return &Extractor{
		fset: token.NewFileSet(),
	}
}

// Language returns "go".
func (e *Extractor) Language() string {
	return Language
}

// Extensions returns Go file extensions.
func (e *Extractor) Extensions() []string {
	return []string{".go"}
}

// CanExtract returns true for .go files.
func (e *Extractor) CanExtract(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".go")
}

// ExtractFile extracts nodes and edges from a single Go file.
func (e *Extractor) ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error) {
	f, err := parser.ParseFile(e.fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	addNode := func(n *graph.Node) {
		nodes = append(nodes, n)
	}

	addEdge := func(edge *graph.Edge) {
		edges = append(edges, edge)
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
			"language":    Language,
		},
	})

	// Extract file node
	fileID := makeID("file", relPath)
	addNode(&graph.Node{
		ID:    fileID,
		Type:  graph.NodeTypeFile,
		Label: filepath.Base(path),
		Attrs: map[string]string{
			"path":     relPath,
			"package":  pkgName,
			"language": Language,
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

		addNode(&graph.Node{
			ID:    importID,
			Type:  graph.NodeTypePackage,
			Label: filepath.Base(importPath),
			Attrs: map[string]string{
				"import_path": importPath,
				"external":    "true",
				"language":    Language,
			},
		})

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
			e.extractFunc(d, relPath, pkgID, fileID, addNode, addEdge)
		case *ast.GenDecl:
			e.extractGenDecl(d, relPath, pkgID, fileID, addNode, addEdge)
		}
	}

	return nodes, edges, nil
}

// DetectFramework returns nil for Go (no framework detection yet).
func (e *Extractor) DetectFramework(path string) *provider.FrameworkInfo {
	// TODO: Could detect gin, echo, chi, etc. via import analysis
	return nil
}

// extractFunc extracts a function or method declaration.
func (e *Extractor) extractFunc(fn *ast.FuncDecl, file, pkgID, fileID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	name := fn.Name.Name
	nodeType := graph.NodeTypeFunction

	// Check if it's a method (has receiver)
	var receiver string
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		nodeType = graph.NodeTypeMethod
		receiver = e.typeString(fn.Recv.List[0].Type)
	}

	// Create unique ID with language prefix
	var funcID string
	if receiver != "" {
		funcID = makeID("method", receiver+"."+name)
	} else {
		funcID = makeID("func", file+"."+name)
	}

	attrs := map[string]string{
		"source_file":     file,
		"source_location": e.position(fn.Pos()),
		"package":         pkgID,
		"language":        Language,
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
				e.extractCall(call, funcID, file, addEdge)
			}
			return true
		})
	}
}

// extractCall extracts a function call.
func (e *Extractor) extractCall(call *ast.CallExpr, callerID, file string, addEdge func(*graph.Edge)) {
	var calleeName string
	var calleeID string

	switch fn := call.Fun.(type) {
	case *ast.Ident:
		calleeName = fn.Name
		calleeID = makeID("func", file+"."+calleeName)
	case *ast.SelectorExpr:
		if x, ok := fn.X.(*ast.Ident); ok {
			calleeName = x.Name + "." + fn.Sel.Name
			calleeID = makeID("call", calleeName)
		}
	default:
		return
	}

	if calleeID == "" {
		return
	}

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

// extractGenDecl extracts type, const, and var declarations.
func (e *Extractor) extractGenDecl(decl *ast.GenDecl, file, pkgID, fileID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			e.extractType(s, decl, file, pkgID, fileID, addNode, addEdge)
		case *ast.ValueSpec:
			e.extractValue(s, decl, file, pkgID, fileID, addNode, addEdge)
		}
	}
}

// extractType extracts a type declaration.
func (e *Extractor) extractType(spec *ast.TypeSpec, decl *ast.GenDecl, file, pkgID, fileID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	name := spec.Name.Name
	typeID := makeID("type", name)

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
		"language":        Language,
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

	addEdge(&graph.Edge{
		From:       fileID,
		To:         typeID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract struct fields and their type references
	if st, ok := spec.Type.(*ast.StructType); ok {
		e.extractStructFields(st, typeID, addEdge)
	}

	// Extract interface methods
	if it, ok := spec.Type.(*ast.InterfaceType); ok {
		e.extractInterfaceMethods(it, typeID, addNode, addEdge)
	}
}

// extractStructFields extracts field type references.
func (e *Extractor) extractStructFields(st *ast.StructType, structID string, addEdge func(*graph.Edge)) {
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

// extractInterfaceMethods extracts method signatures.
func (e *Extractor) extractInterfaceMethods(it *ast.InterfaceType, ifaceID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
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

		for _, name := range method.Names {
			methodID := makeID("method", ifaceID+"."+name.Name)
			addNode(&graph.Node{
				ID:    methodID,
				Type:  graph.NodeTypeMethod,
				Label: name.Name,
				Attrs: map[string]string{
					"interface":       ifaceID,
					"source_location": e.position(method.Pos()),
					"language":        Language,
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

// extractValue extracts const or var declarations.
func (e *Extractor) extractValue(spec *ast.ValueSpec, decl *ast.GenDecl, file, pkgID, fileID string, addNode func(*graph.Node), addEdge func(*graph.Edge)) {
	nodeType := graph.NodeTypeVariable
	if decl.Tok == token.CONST {
		nodeType = graph.NodeTypeConstant
	}

	for _, name := range spec.Names {
		if name.Name == "_" {
			continue
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
				"language":        Language,
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

// makeID creates a stable, filesystem-safe ID with the Go language prefix.
func makeID(prefix, name string) string {
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
	return NodePrefix + prefix + "_" + safe
}

// isBuiltinType returns true for Go builtin types.
func isBuiltinType(name string) bool {
	builtins := map[string]bool{
		"bool": true, "byte": true, "complex64": true, "complex128": true,
		"error": true, "float32": true, "float64": true,
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"rune": true, "string": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"uintptr": true, "any": true,
	}
	return builtins[name]
}

// init registers the Go extractor with the global provider registry.
func init() {
	provider.Register(func() provider.LanguageExtractor {
		return New()
	}, provider.PriorityDefault)
}
