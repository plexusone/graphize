// Package swift provides Swift extraction for knowledge graphs.
package swift

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/swift"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/provider"
)

const (
	// Language is the canonical name for Swift.
	Language = "swift"

	// NodePrefix is the prefix for all Swift node IDs.
	NodePrefix = "swift_"
)

// Extractor implements extract.LanguageExtractor for Swift source code.
type Extractor struct {
	parser *sitter.Parser
}

// New creates a new Swift extractor.
func New() *Extractor {
	parser := sitter.NewParser()
	parser.SetLanguage(swift.GetLanguage())
	return &Extractor{
		parser: parser,
	}
}

// Language returns "swift".
func (e *Extractor) Language() string {
	return Language
}

// Extensions returns Swift file extensions.
func (e *Extractor) Extensions() []string {
	return []string{".swift"}
}

// CanExtract returns true for .swift files.
func (e *Extractor) CanExtract(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".swift")
}

// ExtractFile extracts nodes and edges from a single Swift file.
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

	// Walk the AST
	root := tree.RootNode()
	e.walkTree(root, content, relPath, fileID, &nodes, &edges)

	return nodes, edges, nil
}

// walkTree recursively walks the tree-sitter AST and extracts nodes/edges.
func (e *Extractor) walkTree(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	switch node.Type() {
	case "class_declaration":
		e.extractClass(node, content, file, fileID, nodes, edges)
		return

	case "struct_declaration":
		e.extractStruct(node, content, file, fileID, nodes, edges)
		return

	case "protocol_declaration":
		e.extractProtocol(node, content, file, fileID, nodes, edges)
		return

	case "enum_declaration":
		e.extractEnum(node, content, file, fileID, nodes, edges)
		return

	case "extension_declaration":
		e.extractExtension(node, content, file, fileID, nodes, edges)
		return

	case "function_declaration":
		e.extractFunction(node, content, file, fileID, nodes, edges)

	case "import_declaration":
		e.extractImport(node, content, file, fileID, nodes, edges)
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			e.walkTree(child, content, file, fileID, nodes, edges)
		}
	}
}

// extractClass extracts a class declaration.
func (e *Extractor) extractClass(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = findChildByType(node, "type_identifier")
	}
	if nameNode == nil {
		return
	}

	className := nameNode.Content(content)
	classID := makeID("class", className)

	*nodes = append(*nodes, &graph.Node{
		ID:    classID,
		Type:  graph.NodeTypeClass,
		Label: className,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"language":        Language,
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         classID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract inheritance (superclass and protocols)
	e.extractInheritance(node, content, classID, edges)

	// Extract class body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		bodyNode = findChildByType(node, "class_body")
	}
	if bodyNode != nil {
		e.extractTypeBody(bodyNode, content, file, classID, nodes, edges)
	}
}

// extractStruct extracts a struct declaration.
func (e *Extractor) extractStruct(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = findChildByType(node, "type_identifier")
	}
	if nameNode == nil {
		return
	}

	structName := nameNode.Content(content)
	structID := makeID("struct", structName)

	*nodes = append(*nodes, &graph.Node{
		ID:    structID,
		Type:  graph.NodeTypeStruct,
		Label: structName,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"language":        Language,
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         structID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract protocol conformance
	e.extractInheritance(node, content, structID, edges)

	// Extract struct body
	bodyNode := findChildByType(node, "struct_body")
	if bodyNode != nil {
		e.extractTypeBody(bodyNode, content, file, structID, nodes, edges)
	}
}

// extractProtocol extracts a protocol declaration.
func (e *Extractor) extractProtocol(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = findChildByType(node, "type_identifier")
	}
	if nameNode == nil {
		return
	}

	protocolName := nameNode.Content(content)
	protocolID := makeID("protocol", protocolName)

	*nodes = append(*nodes, &graph.Node{
		ID:    protocolID,
		Type:  graph.NodeTypeInterface, // Protocols are Swift's interfaces
		Label: protocolName,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"language":        Language,
			"kind":            "protocol",
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         protocolID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract protocol inheritance
	e.extractInheritance(node, content, protocolID, edges)

	// Extract protocol body
	bodyNode := findChildByType(node, "protocol_body")
	if bodyNode != nil {
		e.extractTypeBody(bodyNode, content, file, protocolID, nodes, edges)
	}
}

// extractEnum extracts an enum declaration.
func (e *Extractor) extractEnum(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = findChildByType(node, "type_identifier")
	}
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

	// Extract protocol conformance
	e.extractInheritance(node, content, enumID, edges)
}

// extractExtension extracts an extension declaration.
func (e *Extractor) extractExtension(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	// Find the type being extended
	typeNode := findChildByType(node, "type_identifier")
	if typeNode == nil {
		return
	}

	extendedType := typeNode.Content(content)
	extensionID := makeID("extension", extendedType+"_"+file)

	*nodes = append(*nodes, &graph.Node{
		ID:    extensionID,
		Type:  "extension",
		Label: "extension " + extendedType,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"language":        Language,
			"extends":         extendedType,
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         extensionID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Link extension to the type it extends
	extendedID := makeID("class", extendedType) // Could be struct, class, or enum
	*edges = append(*edges, &graph.Edge{
		From:       extensionID,
		To:         extendedID,
		Type:       graph.EdgeTypeExtends,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract extension body
	bodyNode := findChildByType(node, "extension_body")
	if bodyNode != nil {
		e.extractTypeBody(bodyNode, content, file, extensionID, nodes, edges)
	}
}

// extractInheritance extracts inheritance and protocol conformance.
func (e *Extractor) extractInheritance(node *sitter.Node, content []byte, typeID string, edges *[]*graph.Edge) {
	// Look for inheritance clause
	inheritanceNode := findChildByType(node, "inheritance_specifier")
	if inheritanceNode == nil {
		// Also try type_inheritance_clause
		inheritanceNode = findChildByType(node, "type_inheritance_clause")
	}
	if inheritanceNode == nil {
		return
	}

	// Extract all inherited types
	for i := 0; i < int(inheritanceNode.ChildCount()); i++ {
		child := inheritanceNode.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "type_identifier" || child.Type() == "user_type" {
			inheritedName := child.Content(content)
			if inheritedName == "" {
				continue
			}

			// First type is usually superclass (for classes), rest are protocols
			inheritedID := makeID("protocol", inheritedName)
			*edges = append(*edges, &graph.Edge{
				From:       typeID,
				To:         inheritedID,
				Type:       graph.EdgeTypeImplements, // Protocol conformance
				Confidence: graph.ConfidenceExtracted,
			})
		}
	}
}

// extractTypeBody extracts methods and properties from a type body.
func (e *Extractor) extractTypeBody(node *sitter.Node, content []byte, file, typeID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_declaration":
			e.extractMethod(child, content, file, typeID, nodes, edges)

		case "subscript_declaration":
			e.extractMethod(child, content, file, typeID, nodes, edges)

		case "property_declaration", "variable_declaration":
			// Could extract properties if needed
		}
	}
}

// extractMethod extracts a method from a type body.
func (e *Extractor) extractMethod(node *sitter.Node, content []byte, file, typeID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = findChildByType(node, "simple_identifier")
	}
	if nameNode == nil {
		return
	}

	methodName := nameNode.Content(content)
	methodID := makeID("method", typeID+"."+methodName)

	*nodes = append(*nodes, &graph.Node{
		ID:    methodID,
		Type:  graph.NodeTypeMethod,
		Label: methodName,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"parent_type":     typeID,
			"language":        Language,
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       typeID,
		To:         methodID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract calls from method body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		bodyNode = findChildByType(node, "function_body")
	}
	if bodyNode != nil {
		e.extractCalls(bodyNode, content, file, methodID, edges)
	}
}

// extractFunction extracts a top-level function declaration.
func (e *Extractor) extractFunction(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = findChildByType(node, "simple_identifier")
	}
	if nameNode == nil {
		return
	}

	funcName := nameNode.Content(content)
	funcID := makeID("func", file+"."+funcName)

	*nodes = append(*nodes, &graph.Node{
		ID:    funcID,
		Type:  graph.NodeTypeFunction,
		Label: funcName,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"language":        Language,
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         funcID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})

	// Extract calls from function body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		bodyNode = findChildByType(node, "function_body")
	}
	if bodyNode != nil {
		e.extractCalls(bodyNode, content, file, funcID, edges)
	}
}

// extractImport extracts an import declaration.
func (e *Extractor) extractImport(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	// Find the module name
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "identifier" || child.Type() == "simple_identifier") {
			moduleName := child.Content(content)
			moduleID := makeID("module", moduleName)

			*nodes = append(*nodes, &graph.Node{
				ID:    moduleID,
				Type:  graph.NodeTypeModule,
				Label: moduleName,
				Attrs: map[string]string{
					"import_path": moduleName,
					"external":    "true",
					"language":    Language,
				},
			})

			*edges = append(*edges, &graph.Edge{
				From:       fileID,
				To:         moduleID,
				Type:       graph.EdgeTypeImports,
				Confidence: graph.ConfidenceExtracted,
			})
			break
		}
	}
}

// extractCalls recursively finds function calls in a node.
func (e *Extractor) extractCalls(node *sitter.Node, content []byte, file, callerID string, edges *[]*graph.Edge) {
	if node.Type() == "call_expression" {
		// Get the function being called
		funcNode := node.ChildByFieldName("function")
		if funcNode == nil {
			funcNode = findChildByType(node, "simple_identifier")
		}
		if funcNode != nil {
			calleeName := funcNode.Content(content)
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

// DetectFramework returns nil for Swift (no framework detection yet).
func (e *Extractor) DetectFramework(path string) *provider.FrameworkInfo {
	// TODO: Could detect SwiftUI, UIKit, Combine, etc.
	return nil
}

// findChildByType finds the first child with the given type.
func findChildByType(node *sitter.Node, typeName string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == typeName {
			return child
		}
	}
	return nil
}

// makeID creates a stable, filesystem-safe ID with the Swift prefix.
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

// init registers the Swift extractor with the global provider registry.
func init() {
	provider.Register(func() provider.LanguageExtractor {
		return New()
	}, provider.PriorityDefault)
}
