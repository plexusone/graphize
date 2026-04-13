// Package java provides Java extraction for knowledge graphs.
package java

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/provider"
)

const (
	// Language is the canonical name for Java.
	Language = "java"

	// NodePrefix is the prefix for all Java node IDs.
	NodePrefix = "java_"
)

// Extractor implements extract.LanguageExtractor for Java source code.
type Extractor struct {
	parser   *sitter.Parser
	detector *SpringDetector
}

// New creates a new Java extractor.
func New() *Extractor {
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	return &Extractor{
		parser:   parser,
		detector: NewSpringDetector(),
	}
}

// Language returns "java".
func (e *Extractor) Language() string {
	return Language
}

// Extensions returns Java file extensions.
func (e *Extractor) Extensions() []string {
	return []string{".java"}
}

// CanExtract returns true for .java files.
func (e *Extractor) CanExtract(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".java")
}

// ExtractFile extracts nodes and edges from a single Java file.
func (e *Extractor) ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, nil, err
	}
	defer tree.Close()

	relPath, _ := filepath.Rel(baseDir, path)
	if relPath == "" {
		relPath = path
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Add file node
	fileID := makeID("file", relPath)
	nodes = append(nodes, &graph.Node{
		ID:    fileID,
		Type:  graph.NodeTypeFile,
		Label: filepath.Base(path),
		Attrs: map[string]string{
			"path":     relPath,
			"language": Language,
		},
	})

	// Extract package declaration
	var packageName string
	cursor := sitter.NewTreeCursor(tree.RootNode())
	defer cursor.Close()

	root := tree.RootNode()
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child != nil && child.Type() == "package_declaration" {
			packageName = e.extractPackageName(child, content)
			if packageName != "" {
				pkgID := makeID("pkg", packageName)
				nodes = append(nodes, &graph.Node{
					ID:    pkgID,
					Type:  graph.NodeTypePackage,
					Label: packageName,
					Attrs: map[string]string{
						"language": Language,
					},
				})
				edges = append(edges, &graph.Edge{
					From:       fileID,
					To:         pkgID,
					Type:       graph.EdgeTypeContains,
					Confidence: graph.ConfidenceExtracted,
				})
			}
			break
		}
	}

	// Walk the AST for classes, methods, imports
	e.walkTree(root, content, relPath, fileID, packageName, &nodes, &edges)

	return nodes, edges, nil
}

// extractPackageName extracts the package name from a package declaration.
func (e *Extractor) extractPackageName(node *sitter.Node, content []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "scoped_identifier" || child.Type() == "identifier") {
			return child.Content(content)
		}
	}
	return ""
}

// walkTree recursively walks the tree-sitter AST and extracts nodes/edges.
func (e *Extractor) walkTree(node *sitter.Node, content []byte, file, fileID, packageName string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	switch node.Type() {
	case "class_declaration":
		e.extractClass(node, content, file, fileID, packageName, nodes, edges)
		return // Don't recurse into class body again

	case "interface_declaration":
		e.extractInterface(node, content, file, fileID, packageName, nodes, edges)
		return

	case "enum_declaration":
		e.extractEnum(node, content, file, fileID, nodes, edges)
		return

	case "import_declaration":
		e.extractImport(node, content, file, fileID, nodes, edges)
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			e.walkTree(child, content, file, fileID, packageName, nodes, edges)
		}
	}
}

// extractClass extracts a class declaration.
func (e *Extractor) extractClass(node *sitter.Node, content []byte, file, fileID, packageName string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	className := nameNode.Content(content)
	classID := makeID("class", className)

	attrs := map[string]string{
		"source_file":     file,
		"source_location": formatLocation(file, node),
		"language":        Language,
	}
	if packageName != "" {
		attrs["package"] = packageName
	}

	// Check for annotations (Spring, etc.)
	annotations := e.extractAnnotations(node, content)
	if len(annotations) > 0 {
		attrs["annotations"] = strings.Join(annotations, ",")
	}

	// Spring layer detection
	layer := e.detector.DetectLayer(annotations)
	if layer != "" {
		attrs["layer"] = layer
	}

	*nodes = append(*nodes, &graph.Node{
		ID:    classID,
		Type:  graph.NodeTypeClass,
		Label: className,
		Attrs: attrs,
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         classID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Add annotation edges
	for _, ann := range annotations {
		annID := makeID("annotation", ann)
		*nodes = append(*nodes, &graph.Node{
			ID:    annID,
			Type:  "annotation",
			Label: ann,
			Attrs: map[string]string{
				"language": Language,
			},
		})
		*edges = append(*edges, &graph.Edge{
			From:       classID,
			To:         annID,
			Type:       graph.EdgeTypeAnnotatedWith,
			Confidence: graph.ConfidenceExtracted,
		})
	}

	// Check for superclass
	superclassNode := node.ChildByFieldName("superclass")
	if superclassNode != nil {
		superName := e.extractTypeName(superclassNode, content)
		if superName != "" {
			superID := makeID("class", superName)
			*edges = append(*edges, &graph.Edge{
				From:       classID,
				To:         superID,
				Type:       graph.EdgeTypeExtends,
				Confidence: graph.ConfidenceExtracted,
			})
		}
	}

	// Check for interfaces
	interfacesNode := node.ChildByFieldName("interfaces")
	if interfacesNode != nil {
		e.extractImplementedInterfaces(interfacesNode, content, classID, edges)
	}

	// Extract class body (fields and methods)
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		e.extractClassBody(bodyNode, content, file, classID, nodes, edges)
	}
}

// extractAnnotations extracts annotations from a node.
func (e *Extractor) extractAnnotations(node *sitter.Node, content []byte) []string {
	var annotations []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "modifiers" {
			for j := 0; j < int(child.ChildCount()); j++ {
				mod := child.Child(j)
				if mod != nil && mod.Type() == "marker_annotation" || mod.Type() == "annotation" {
					annName := e.extractAnnotationName(mod, content)
					if annName != "" {
						annotations = append(annotations, annName)
					}
				}
			}
		}
	}

	return annotations
}

// extractAnnotationName gets the name of an annotation.
func (e *Extractor) extractAnnotationName(node *sitter.Node, content []byte) string {
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		return nameNode.Content(content)
	}
	// For marker annotations without explicit name field
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "identifier" {
			return child.Content(content)
		}
	}
	return ""
}

// extractTypeName extracts a type name from various type nodes.
func (e *Extractor) extractTypeName(node *sitter.Node, content []byte) string {
	switch node.Type() {
	case "type_identifier", "identifier":
		return node.Content(content)
	case "generic_type":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "type_identifier" {
				return child.Content(content)
			}
		}
	}

	// Try to find type_identifier child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			name := e.extractTypeName(child, content)
			if name != "" {
				return name
			}
		}
	}
	return ""
}

// extractImplementedInterfaces extracts implemented interface relationships.
func (e *Extractor) extractImplementedInterfaces(node *sitter.Node, content []byte, classID string, edges *[]*graph.Edge) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			ifaceName := e.extractTypeName(child, content)
			if ifaceName != "" {
				ifaceID := makeID("interface", ifaceName)
				*edges = append(*edges, &graph.Edge{
					From:       classID,
					To:         ifaceID,
					Type:       graph.EdgeTypeImplements,
					Confidence: graph.ConfidenceExtracted,
				})
			}
		}
	}
}

// extractClassBody extracts methods and fields from a class body.
func (e *Extractor) extractClassBody(node *sitter.Node, content []byte, file, classID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_declaration", "constructor_declaration":
			e.extractMethod(child, content, file, classID, nodes, edges)

		case "field_declaration":
			e.extractField(child, content, file, classID, nodes, edges)
		}
	}
}

// extractMethod extracts a method declaration.
func (e *Extractor) extractMethod(node *sitter.Node, content []byte, file, classID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	methodName := nameNode.Content(content)
	methodID := makeID("method", classID+"."+methodName)

	attrs := map[string]string{
		"source_file":     file,
		"source_location": formatLocation(file, node),
		"class":           classID,
		"language":        Language,
	}

	// Check for annotations
	annotations := e.extractAnnotations(node, content)
	if len(annotations) > 0 {
		attrs["annotations"] = strings.Join(annotations, ",")
	}

	// Check for Spring request mappings
	route := e.detector.DetectRoute(annotations, node, content)
	if route != "" {
		attrs["route"] = route
	}

	*nodes = append(*nodes, &graph.Node{
		ID:    methodID,
		Type:  graph.NodeTypeMethod,
		Label: methodName,
		Attrs: attrs,
	})

	*edges = append(*edges, &graph.Edge{
		From:       classID,
		To:         methodID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Add route edge if present
	if route != "" {
		routeID := makeID("route", route)
		*nodes = append(*nodes, &graph.Node{
			ID:    routeID,
			Type:  "route",
			Label: route,
			Attrs: map[string]string{
				"language": Language,
			},
		})
		*edges = append(*edges, &graph.Edge{
			From:       methodID,
			To:         routeID,
			Type:       graph.EdgeTypeHandlesRoute,
			Confidence: graph.ConfidenceExtracted,
		})
	}

	// Add annotation edges for method
	for _, ann := range annotations {
		annID := makeID("annotation", ann)
		*edges = append(*edges, &graph.Edge{
			From:       methodID,
			To:         annID,
			Type:       graph.EdgeTypeAnnotatedWith,
			Confidence: graph.ConfidenceExtracted,
		})
	}

	// Extract calls from method body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		e.extractCalls(bodyNode, content, file, methodID, edges)
	}
}

// extractField extracts a field declaration and detects injection.
func (e *Extractor) extractField(node *sitter.Node, content []byte, file, classID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	annotations := e.extractAnnotations(node, content)

	// Check for dependency injection
	if e.detector.IsInjection(annotations) {
		// Find the type being injected
		typeNode := node.ChildByFieldName("type")
		if typeNode != nil {
			typeName := e.extractTypeName(typeNode, content)
			if typeName != "" {
				typeID := makeID("class", typeName)
				*edges = append(*edges, &graph.Edge{
					From:       classID,
					To:         typeID,
					Type:       graph.EdgeTypeInjects,
					Confidence: graph.ConfidenceExtracted,
					Attrs: map[string]string{
						"injection_type": strings.Join(annotations, ","),
					},
				})
			}
		}
	}
}

// extractInterface extracts an interface declaration.
func (e *Extractor) extractInterface(node *sitter.Node, content []byte, file, fileID, packageName string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	ifaceName := nameNode.Content(content)
	ifaceID := makeID("interface", ifaceName)

	attrs := map[string]string{
		"source_file":     file,
		"source_location": formatLocation(file, node),
		"language":        Language,
	}
	if packageName != "" {
		attrs["package"] = packageName
	}

	*nodes = append(*nodes, &graph.Node{
		ID:    ifaceID,
		Type:  graph.NodeTypeInterface,
		Label: ifaceName,
		Attrs: attrs,
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         ifaceID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})
}

// extractEnum extracts an enum declaration.
func (e *Extractor) extractEnum(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	enumName := nameNode.Content(content)
	enumID := makeID("enum", enumName)

	*nodes = append(*nodes, &graph.Node{
		ID:    enumID,
		Type:  "enum",
		Label: enumName,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"language":        Language,
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         enumID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})
}

// extractImport extracts an import declaration.
func (e *Extractor) extractImport(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "scoped_identifier" || child.Type() == "identifier") {
			importPath := child.Content(content)
			importID := makeID("pkg", importPath)

			*nodes = append(*nodes, &graph.Node{
				ID:    importID,
				Type:  graph.NodeTypePackage,
				Label: filepath.Base(importPath),
				Attrs: map[string]string{
					"import_path": importPath,
					"external":    "true",
					"language":    Language,
				},
			})

			*edges = append(*edges, &graph.Edge{
				From:       fileID,
				To:         importID,
				Type:       graph.EdgeTypeImports,
				Confidence: graph.ConfidenceExtracted,
			})
			break
		}
	}
}

// extractCalls recursively finds method calls in a node.
func (e *Extractor) extractCalls(node *sitter.Node, content []byte, file, callerID string, edges *[]*graph.Edge) {
	if node.Type() == "method_invocation" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			calleeName := nameNode.Content(content)

			// Try to get the object being called on
			objNode := node.ChildByFieldName("object")
			if objNode != nil {
				calleeName = objNode.Content(content) + "." + calleeName
			}

			calleeID := makeID("call", calleeName)

			*edges = append(*edges, &graph.Edge{
				From:       callerID,
				To:         calleeID,
				Type:       graph.EdgeTypeCalls,
				Confidence: graph.ConfidenceExtracted,
				Attrs: map[string]string{
					"source_location": formatLocation(file, node),
					"callee_name":     calleeName,
				},
			})
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			e.extractCalls(child, content, file, callerID, edges)
		}
	}
}

// DetectFramework returns Spring framework info if detected.
func (e *Extractor) DetectFramework(path string) *provider.FrameworkInfo {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil
	}
	defer tree.Close()

	return e.detector.DetectSpring(tree.RootNode(), content)
}

// makeID creates a stable, filesystem-safe ID with the Java prefix.
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

// formatLocation formats a source location string.
func formatLocation(file string, node *sitter.Node) string {
	start := node.StartPoint()
	return file + ":" + itoa(int(start.Row)+1) + ":" + itoa(int(start.Column)+1)
}

// itoa converts an int to string.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// init registers the Java extractor with the global provider registry.
func init() {
	provider.Register(func() provider.LanguageExtractor {
		return New()
	}, provider.PriorityDefault)
}
