// Package typescript provides TypeScript/JavaScript extraction for knowledge graphs.
package typescript

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/provider"
)

const (
	// Language is the canonical name for TypeScript.
	Language = "typescript"

	// NodePrefix is the prefix for all TypeScript node IDs.
	NodePrefix = "ts_"
)

// Extractor implements extract.LanguageExtractor for TypeScript source code.
type Extractor struct {
	parser *sitter.Parser
}

// New creates a new TypeScript extractor.
func New() *Extractor {
	parser := sitter.NewParser()
	parser.SetLanguage(typescript.GetLanguage())
	return &Extractor{
		parser: parser,
	}
}

// Language returns "typescript".
func (e *Extractor) Language() string {
	return Language
}

// Extensions returns TypeScript/JavaScript file extensions.
func (e *Extractor) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx"}
}

// CanExtract returns true for TypeScript/JavaScript files.
func (e *Extractor) CanExtract(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supported := range e.Extensions() {
		if ext == supported {
			return true
		}
	}
	return false
}

// ExtractFile extracts nodes and edges from a single TypeScript file.
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
	cursor := sitter.NewTreeCursor(tree.RootNode())
	defer cursor.Close()

	e.walkTree(cursor, content, relPath, fileID, &nodes, &edges)

	return nodes, edges, nil
}

// walkTree recursively walks the tree-sitter AST and extracts nodes/edges.
func (e *Extractor) walkTree(cursor *sitter.TreeCursor, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	node := cursor.CurrentNode()

	switch node.Type() {
	case "class_declaration", "class":
		e.extractClass(node, content, file, fileID, nodes, edges)

	case "function_declaration", "function":
		e.extractFunction(node, content, file, fileID, nodes, edges)

	case "arrow_function", "function_expression":
		// Skip anonymous functions at top level, they'll be captured as method definitions

	case "method_definition":
		// Methods are extracted as part of class extraction

	case "interface_declaration":
		e.extractInterface(node, content, file, fileID, nodes, edges)

	case "type_alias_declaration":
		e.extractTypeAlias(node, content, file, fileID, nodes, edges)

	case "import_statement":
		e.extractImport(node, content, file, fileID, nodes, edges)

	case "export_statement":
		// Handle exports if needed
	}

	// Recurse into children
	if cursor.GoToFirstChild() {
		for {
			e.walkTree(cursor, content, file, fileID, nodes, edges)
			if !cursor.GoToNextSibling() {
				break
			}
		}
		cursor.GoToParent()
	}
}

// extractClass extracts a class declaration.
func (e *Extractor) extractClass(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
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

	// Check for heritage clause (extends, implements)
	heritageNode := findChildByType(node, "class_heritage")
	if heritageNode != nil {
		e.extractHeritage(heritageNode, content, classID, nodes, edges)
	}

	// Extract methods
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		e.extractClassBody(bodyNode, content, file, classID, nodes, edges)
	}
}

// extractHeritage extracts extends/implements relationships.
func (e *Extractor) extractHeritage(node *sitter.Node, content []byte, classID string, _ *[]*graph.Node, edges *[]*graph.Edge) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "extends_clause":
			// Get the extended type
			for j := 0; j < int(child.ChildCount()); j++ {
				typeNode := child.Child(j)
				if typeNode != nil && typeNode.Type() == "identifier" {
					parentType := typeNode.Content(content)
					parentID := makeID("class", parentType)
					*edges = append(*edges, &graph.Edge{
						From:       classID,
						To:         parentID,
						Type:       graph.EdgeTypeExtends,
						Confidence: graph.ConfidenceExtracted,
					})
				}
			}

		case "implements_clause":
			// Get implemented interfaces
			for j := 0; j < int(child.ChildCount()); j++ {
				typeNode := child.Child(j)
				if typeNode != nil && typeNode.Type() == "type_identifier" {
					ifaceName := typeNode.Content(content)
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
}

// extractClassBody extracts methods and properties from a class body.
func (e *Extractor) extractClassBody(node *sitter.Node, content []byte, file, classID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "method_definition":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				methodName := nameNode.Content(content)
				methodID := makeID("method", classID+"."+methodName)

				*nodes = append(*nodes, &graph.Node{
					ID:    methodID,
					Type:  graph.NodeTypeMethod,
					Label: methodName,
					Attrs: map[string]string{
						"source_file":     file,
						"source_location": formatLocation(file, child),
						"class":           classID,
						"language":        Language,
					},
				})

				*edges = append(*edges, &graph.Edge{
					From:       classID,
					To:         methodID,
					Type:       graph.EdgeTypeContains,
					Confidence: graph.ConfidenceExtracted,
				})

				// Extract calls from method body
				bodyNode := child.ChildByFieldName("body")
				if bodyNode != nil {
					e.extractCalls(bodyNode, content, file, methodID, edges)
				}
			}

		case "public_field_definition", "field_definition":
			// Could extract class properties here if needed
		}
	}
}

// extractFunction extracts a function declaration.
func (e *Extractor) extractFunction(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
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
	if bodyNode != nil {
		e.extractCalls(bodyNode, content, file, funcID, edges)
	}
}

// extractInterface extracts an interface declaration.
func (e *Extractor) extractInterface(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	ifaceName := nameNode.Content(content)
	ifaceID := makeID("interface", ifaceName)

	*nodes = append(*nodes, &graph.Node{
		ID:    ifaceID,
		Type:  graph.NodeTypeInterface,
		Label: ifaceName,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"language":        Language,
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         ifaceID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})
}

// extractTypeAlias extracts a type alias declaration.
func (e *Extractor) extractTypeAlias(node *sitter.Node, content []byte, file, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	typeName := nameNode.Content(content)
	typeID := makeID("type", typeName)

	*nodes = append(*nodes, &graph.Node{
		ID:    typeID,
		Type:  "type",
		Label: typeName,
		Attrs: map[string]string{
			"source_file":     file,
			"source_location": formatLocation(file, node),
			"language":        Language,
		},
	})

	*edges = append(*edges, &graph.Edge{
		From:       fileID,
		To:         typeID,
		Type:       graph.EdgeTypeContains,
		Confidence: graph.ConfidenceExtracted,
	})
}

// extractImport extracts an import statement.
func (e *Extractor) extractImport(node *sitter.Node, content []byte, _, fileID string, nodes *[]*graph.Node, edges *[]*graph.Edge) {
	sourceNode := node.ChildByFieldName("source")
	if sourceNode == nil {
		return
	}

	// Get the module path (strip quotes)
	modulePath := strings.Trim(sourceNode.Content(content), `"'`)
	moduleID := makeID("module", modulePath)

	*nodes = append(*nodes, &graph.Node{
		ID:    moduleID,
		Type:  graph.NodeTypeModule,
		Label: filepath.Base(modulePath),
		Attrs: map[string]string{
			"import_path": modulePath,
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
}

// extractCalls recursively finds function calls in a node.
func (e *Extractor) extractCalls(node *sitter.Node, content []byte, file, callerID string, edges *[]*graph.Edge) {
	if node.Type() == "call_expression" {
		funcNode := node.ChildByFieldName("function")
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

// DetectFramework returns framework information if detected.
func (e *Extractor) DetectFramework(path string) *provider.FrameworkInfo {
	// TODO: Detect Express, React, Angular, Vue, etc.
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

// makeID creates a stable, filesystem-safe ID with the TypeScript prefix.
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

// itoa converts an int to string without importing strconv.
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

// init registers the TypeScript extractor with the global provider registry.
func init() {
	provider.Register(func() provider.LanguageExtractor {
		return New()
	}, provider.PriorityDefault)
}
