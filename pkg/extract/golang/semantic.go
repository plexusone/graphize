// Package golang provides Go language extraction for knowledge graphs.
// This file implements semantic analysis using go/types for enhanced extraction.
package golang

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/provider"
	"golang.org/x/tools/go/packages"
)

// SemanticExtractor provides enhanced Go extraction with type information.
// It uses golang.org/x/tools/go/packages for full semantic analysis.
type SemanticExtractor struct {
	// baseDir is the root directory for the module being analyzed.
	baseDir string

	// pkgCache caches loaded packages.
	pkgCache map[string]*packages.Package

	// fset is the shared file set for all packages.
	fset *token.FileSet

	// typeInfo maps file paths to their type information.
	typeInfo map[string]*types.Info

	// enableFrameworkDetection enables detection of Go web frameworks.
	enableFrameworkDetection bool
}

// SemanticOption configures the SemanticExtractor.
type SemanticOption func(*SemanticExtractor)

// WithFrameworkDetection enables framework detection.
func WithFrameworkDetection(enable bool) SemanticOption {
	return func(e *SemanticExtractor) {
		e.enableFrameworkDetection = enable
	}
}

// NewSemanticExtractor creates a new semantic-aware Go extractor.
func NewSemanticExtractor(opts ...SemanticOption) *SemanticExtractor {
	e := &SemanticExtractor{
		pkgCache:                 make(map[string]*packages.Package),
		fset:                     token.NewFileSet(),
		typeInfo:                 make(map[string]*types.Info),
		enableFrameworkDetection: true,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Language returns "go".
func (e *SemanticExtractor) Language() string {
	return Language
}

// Extensions returns Go file extensions.
func (e *SemanticExtractor) Extensions() []string {
	return []string{".go"}
}

// CanExtract returns true for .go files.
func (e *SemanticExtractor) CanExtract(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".go")
}

// LoadPackage loads a Go package with full type information.
func (e *SemanticExtractor) LoadPackage(dir string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesSizes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedDeps,
		Dir:  dir,
		Fset: e.fset,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("loading packages: %w", err)
	}

	// Cache packages and their type info
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			// Log but don't fail - partial analysis is still useful
			continue
		}
		e.pkgCache[pkg.PkgPath] = pkg
		for _, file := range pkg.GoFiles {
			e.typeInfo[file] = pkg.TypesInfo
		}
	}

	e.baseDir = dir
	return nil
}

// ExtractFile extracts nodes and edges from a single Go file with semantic analysis.
func (e *SemanticExtractor) ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error) {
	// Find the package containing this file
	var pkg *packages.Package
	var astFile *ast.File

	absPath, _ := filepath.Abs(path)
	for _, p := range e.pkgCache {
		for i, f := range p.GoFiles {
			if f == absPath || f == path {
				pkg = p
				if i < len(p.Syntax) {
					astFile = p.Syntax[i]
				}
				break
			}
		}
		if pkg != nil {
			break
		}
	}

	// Fall back to basic extraction if package not loaded
	if pkg == nil || astFile == nil {
		basic := New()
		return basic.ExtractFile(path, baseDir)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	addNode := func(n *graph.Node) { nodes = append(nodes, n) }
	addEdge := func(edge *graph.Edge) { edges = append(edges, edge) }

	relPath, _ := filepath.Rel(baseDir, path)
	if relPath == "" {
		relPath = path
	}

	// Extract package node
	pkgName := pkg.Name
	pkgPath := pkg.PkgPath
	pkgID := makeID("pkg", pkgPath)
	addNode(&graph.Node{
		ID:    pkgID,
		Type:  graph.NodeTypePackage,
		Label: pkgName,
		Attrs: map[string]string{
			"source_file": relPath,
			"language":    Language,
			"import_path": pkgPath,
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

	addEdge(&graph.Edge{
		From:       fileID,
		To:         pkgID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract imports with resolved paths
	for _, imp := range astFile.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		importID := makeID("pkg", importPath)

		attrs := map[string]string{
			"import_path": importPath,
			"language":    Language,
		}

		// Check if it's a standard library package
		if isStdLib(importPath) {
			attrs["stdlib"] = "true"
		} else {
			attrs["external"] = "true"
		}

		addNode(&graph.Node{
			ID:    importID,
			Type:  graph.NodeTypePackage,
			Label: filepath.Base(importPath),
			Attrs: attrs,
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

	// Extract declarations with type information
	for _, decl := range astFile.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			e.extractSemanticFunc(d, pkg, relPath, pkgID, fileID, addNode, addEdge)
		case *ast.GenDecl:
			e.extractSemanticGenDecl(d, pkg, relPath, pkgID, fileID, addNode, addEdge)
		}
	}

	return nodes, edges, nil
}

// extractSemanticFunc extracts a function with full type resolution.
func (e *SemanticExtractor) extractSemanticFunc(
	fn *ast.FuncDecl,
	pkg *packages.Package,
	file, pkgID, fileID string,
	addNode func(*graph.Node),
	addEdge func(*graph.Edge),
) {
	name := fn.Name.Name
	nodeType := graph.NodeTypeFunction

	// Get the types.Func object for this declaration
	obj := pkg.TypesInfo.Defs[fn.Name]
	if obj == nil {
		// Fall back to basic extraction
		basic := &Extractor{fset: e.fset}
		basic.extractFunc(fn, file, pkgID, fileID, addNode, addEdge)
		return
	}

	funcObj, ok := obj.(*types.Func)
	if !ok {
		return
	}

	// Get signature information
	sig := funcObj.Type().(*types.Signature)

	// Check if it's a method
	var receiver string
	var receiverType types.Type
	if sig.Recv() != nil {
		nodeType = graph.NodeTypeMethod
		receiverType = sig.Recv().Type()
		receiver = typeNameString(receiverType)
	}

	// Create unique ID using fully qualified name
	var funcID string
	fullName := funcObj.FullName()
	if receiver != "" {
		funcID = makeID("method", receiver+"."+name)
	} else {
		funcID = makeID("func", pkg.PkgPath+"."+name)
	}

	attrs := map[string]string{
		"source_file":     file,
		"source_location": e.position(fn.Pos()),
		"package":         pkgID,
		"language":        Language,
		"full_name":       fullName,
		"signature":       sig.String(),
	}

	if receiver != "" {
		attrs["receiver"] = receiver
	}
	if fn.Doc != nil {
		attrs["doc"] = fn.Doc.Text()
	}

	// Detect if this is an HTTP handler
	if e.enableFrameworkDetection {
		if handlerType := detectHandlerType(sig); handlerType != "" {
			attrs["handler_type"] = handlerType
			attrs["is_entrypoint"] = "true"
		}
	}

	// Check if exported
	if ast.IsExported(name) {
		attrs["exported"] = "true"
	}

	addNode(&graph.Node{
		ID:    funcID,
		Type:  nodeType,
		Label: name,
		Attrs: attrs,
	})

	addEdge(&graph.Edge{
		From:       fileID,
		To:         funcID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Link method to receiver type
	if receiver != "" {
		typeID := makeID("type", receiver)
		addEdge(&graph.Edge{
			From:       funcID,
			To:         typeID,
			Type:       "method_of",
			Confidence: graph.ConfidenceExtracted,
		})
	}

	// Extract function calls with type resolution
	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				e.extractSemanticCall(call, funcID, pkg, addEdge)
			}
			return true
		})
	}
}

// extractSemanticCall extracts a function call with full type resolution.
func (e *SemanticExtractor) extractSemanticCall(
	call *ast.CallExpr,
	callerID string,
	pkg *packages.Package,
	addEdge func(*graph.Edge),
) {
	// Try to resolve the called function
	var calleeName string
	var calleeID string
	var calleePkg string

	switch fn := call.Fun.(type) {
	case *ast.Ident:
		// Local function call or builtin
		if obj := pkg.TypesInfo.Uses[fn]; obj != nil {
			if funcObj, ok := obj.(*types.Func); ok {
				calleeName = funcObj.Name()
				calleePkg = funcObj.Pkg().Path()
				calleeID = makeID("func", calleePkg+"."+calleeName)
			} else if builtin, ok := obj.(*types.Builtin); ok {
				calleeName = builtin.Name()
				calleeID = makeID("builtin", calleeName)
			}
		}

	case *ast.SelectorExpr:
		// Method call or qualified function call
		if sel := pkg.TypesInfo.Selections[fn]; sel != nil {
			// Method call
			if funcObj := sel.Obj(); funcObj != nil {
				if f, ok := funcObj.(*types.Func); ok {
					recv := sel.Recv()
					recvName := typeNameString(recv)
					calleeName = recvName + "." + f.Name()
					calleeID = makeID("method", calleeName)
					if f.Pkg() != nil {
						calleePkg = f.Pkg().Path()
					}
				}
			}
		} else if obj := pkg.TypesInfo.Uses[fn.Sel]; obj != nil {
			// Qualified function call (pkg.Func)
			if funcObj, ok := obj.(*types.Func); ok {
				calleeName = funcObj.Name()
				if funcObj.Pkg() != nil {
					calleePkg = funcObj.Pkg().Path()
					calleeID = makeID("func", calleePkg+"."+calleeName)
				}
			}
		}
	}

	if calleeID == "" {
		return
	}

	attrs := map[string]string{
		"source_location": e.position(call.Pos()),
		"callee_name":     calleeName,
	}
	if calleePkg != "" {
		attrs["callee_package"] = calleePkg
	}

	addEdge(&graph.Edge{
		From:       callerID,
		To:         calleeID,
		Type:       graph.EdgeTypeCalls,
		Confidence: graph.ConfidenceExtracted,
		Attrs:      attrs,
	})
}

// extractSemanticGenDecl extracts type, const, and var declarations with type info.
func (e *SemanticExtractor) extractSemanticGenDecl(
	decl *ast.GenDecl,
	pkg *packages.Package,
	file, pkgID, fileID string,
	addNode func(*graph.Node),
	addEdge func(*graph.Edge),
) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			e.extractSemanticType(s, decl, pkg, file, pkgID, fileID, addNode, addEdge)
		case *ast.ValueSpec:
			e.extractSemanticValue(s, decl, pkg, file, pkgID, fileID, addNode, addEdge)
		}
	}
}

// extractSemanticType extracts a type declaration with interface implementation detection.
func (e *SemanticExtractor) extractSemanticType(
	spec *ast.TypeSpec,
	decl *ast.GenDecl,
	pkg *packages.Package,
	file, pkgID, fileID string,
	addNode func(*graph.Node),
	addEdge func(*graph.Edge),
) {
	name := spec.Name.Name
	obj := pkg.TypesInfo.Defs[spec.Name]
	if obj == nil {
		// Fall back to basic extraction
		basic := &Extractor{fset: e.fset}
		basic.extractType(spec, decl, file, pkgID, fileID, addNode, addEdge)
		return
	}

	typeObj, ok := obj.(*types.TypeName)
	if !ok {
		return
	}

	typeID := makeID("type", pkg.PkgPath+"."+name)
	nodeType := "type"

	underlying := typeObj.Type().Underlying()
	switch underlying.(type) {
	case *types.Struct:
		nodeType = graph.NodeTypeStruct
	case *types.Interface:
		nodeType = graph.NodeTypeInterface
	}

	attrs := map[string]string{
		"source_file":     file,
		"source_location": e.position(spec.Pos()),
		"package":         pkgID,
		"language":        Language,
		"full_name":       pkg.PkgPath + "." + name,
	}

	if ast.IsExported(name) {
		attrs["exported"] = "true"
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

	// Detect interface implementations
	if _, isStruct := underlying.(*types.Struct); isStruct {
		e.detectInterfaceImplementations(typeObj.Type(), typeID, pkg, addEdge)
	}

	// Extract struct fields
	if st, ok := spec.Type.(*ast.StructType); ok {
		e.extractSemanticStructFields(st, typeID, pkg, addEdge)
	}

	// Extract interface methods
	if it, ok := spec.Type.(*ast.InterfaceType); ok {
		basic := &Extractor{fset: e.fset}
		basic.extractInterfaceMethods(it, typeID, addNode, addEdge)
	}
}

// detectInterfaceImplementations finds interfaces that a type implements.
func (e *SemanticExtractor) detectInterfaceImplementations(
	t types.Type,
	typeID string,
	_ *packages.Package, // pkg unused - we check all cached packages
	addEdge func(*graph.Edge),
) {
	// Check against interfaces in the same package and imported packages
	for _, p := range e.pkgCache {
		scope := p.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if typeName, ok := obj.(*types.TypeName); ok {
				if iface, ok := typeName.Type().Underlying().(*types.Interface); ok {
					// Check if t implements iface
					if types.Implements(t, iface) || types.Implements(types.NewPointer(t), iface) {
						ifaceID := makeID("type", p.PkgPath+"."+name)
						addEdge(&graph.Edge{
							From:       typeID,
							To:         ifaceID,
							Type:       graph.EdgeTypeImplements,
							Confidence: graph.ConfidenceExtracted,
							Attrs: map[string]string{
								"interface_package": p.PkgPath,
							},
						})
					}
				}
			}
		}
	}
}

// extractSemanticStructFields extracts field type references with full resolution.
func (e *SemanticExtractor) extractSemanticStructFields(
	st *ast.StructType,
	structID string,
	pkg *packages.Package,
	addEdge func(*graph.Edge),
) {
	if st.Fields == nil {
		return
	}

	for _, field := range st.Fields.List {
		// Get the type of the field
		tv := pkg.TypesInfo.Types[field.Type]
		if !tv.IsType() {
			continue
		}

		typeName := typeNameString(tv.Type)
		if typeName == "" || isBuiltinType(typeName) {
			continue
		}

		// Try to get the full package path
		if named, ok := tv.Type.(*types.Named); ok {
			if obj := named.Obj(); obj != nil && obj.Pkg() != nil {
				typeName = obj.Pkg().Path() + "." + obj.Name()
			}
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

// extractSemanticValue extracts const or var with type info.
func (e *SemanticExtractor) extractSemanticValue(
	spec *ast.ValueSpec,
	decl *ast.GenDecl,
	_ *packages.Package, // pkg unused - using basic extraction for now
	file, pkgID, fileID string,
	addNode func(*graph.Node),
	addEdge func(*graph.Edge),
) {
	// Use basic extraction for now
	basic := &Extractor{fset: e.fset}
	basic.extractValue(spec, decl, file, pkgID, fileID, addNode, addEdge)
}

// DetectFramework detects Go web frameworks.
func (e *SemanticExtractor) DetectFramework(path string) *provider.FrameworkInfo {
	if !e.enableFrameworkDetection {
		return nil
	}

	// Check imports for known frameworks
	for _, pkg := range e.pkgCache {
		for importPath := range pkg.Imports {
			if info := detectFrameworkFromImport(importPath); info != nil {
				return info
			}
		}
	}

	return nil
}

// position returns a string representation of a token position.
func (e *SemanticExtractor) position(pos token.Pos) string {
	p := e.fset.Position(pos)
	return p.String()
}

// typeNameString returns a clean string representation of a type.
func typeNameString(t types.Type) string {
	switch typ := t.(type) {
	case *types.Named:
		if obj := typ.Obj(); obj != nil {
			return obj.Name()
		}
	case *types.Pointer:
		return typeNameString(typ.Elem())
	case *types.Slice:
		return typeNameString(typ.Elem())
	case *types.Array:
		return typeNameString(typ.Elem())
	case *types.Map:
		return "map"
	case *types.Chan:
		return "chan"
	case *types.Basic:
		return typ.Name()
	}
	return ""
}

// detectHandlerType checks if a function signature matches known handler patterns.
func detectHandlerType(sig *types.Signature) string {
	params := sig.Params()

	// Check for http.HandlerFunc pattern: func(http.ResponseWriter, *http.Request)
	if params.Len() == 2 {
		p0 := params.At(0).Type().String()
		p1 := params.At(1).Type().String()
		if strings.Contains(p0, "http.ResponseWriter") && strings.Contains(p1, "http.Request") {
			return "http.HandlerFunc"
		}
	}

	// Check for gin.HandlerFunc pattern: func(*gin.Context)
	if params.Len() == 1 {
		p0 := params.At(0).Type().String()
		if strings.Contains(p0, "gin.Context") {
			return "gin.HandlerFunc"
		}
		if strings.Contains(p0, "echo.Context") {
			return "echo.HandlerFunc"
		}
		if strings.Contains(p0, "fiber.Ctx") {
			return "fiber.Handler"
		}
	}

	return ""
}

// detectFrameworkFromImport detects framework from import path.
func detectFrameworkFromImport(importPath string) *provider.FrameworkInfo {
	frameworks := map[string]*provider.FrameworkInfo{
		"github.com/gin-gonic/gin": {
			Name:  "Gin",
			Layer: "controller",
		},
		"github.com/labstack/echo": {
			Name:  "Echo",
			Layer: "controller",
		},
		"github.com/gofiber/fiber": {
			Name:  "Fiber",
			Layer: "controller",
		},
		"github.com/gorilla/mux": {
			Name:  "Gorilla Mux",
			Layer: "controller",
		},
		"github.com/go-chi/chi": {
			Name:  "Chi",
			Layer: "controller",
		},
		"google.golang.org/grpc": {
			Name:  "gRPC",
			Layer: "service",
		},
	}

	for prefix, info := range frameworks {
		if strings.HasPrefix(importPath, prefix) {
			return info
		}
	}

	return nil
}

// isStdLib checks if an import path is from the Go standard library.
func isStdLib(importPath string) bool {
	// Standard library packages don't contain dots in the first segment
	if strings.Contains(importPath, ".") {
		return false
	}
	// Check for known stdlib prefixes
	stdPrefixes := []string{
		"archive", "bufio", "bytes", "compress", "container", "context",
		"crypto", "database", "debug", "embed", "encoding", "errors",
		"expvar", "flag", "fmt", "go", "hash", "html", "image", "index",
		"io", "log", "maps", "math", "mime", "net", "os", "path", "plugin",
		"reflect", "regexp", "runtime", "slices", "sort", "strconv", "strings",
		"sync", "syscall", "testing", "text", "time", "unicode", "unsafe",
	}
	first := strings.Split(importPath, "/")[0]
	for _, prefix := range stdPrefixes {
		if first == prefix {
			return true
		}
	}
	return false
}
