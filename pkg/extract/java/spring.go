package java

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/plexusone/graphize/provider"
)

// SpringDetector detects Spring framework annotations and patterns.
type SpringDetector struct {
	// layerAnnotations maps annotations to architectural layers
	layerAnnotations map[string]string

	// injectionAnnotations are annotations that indicate dependency injection
	injectionAnnotations map[string]bool

	// routeAnnotations are annotations that map to HTTP routes
	routeAnnotations map[string]bool
}

// NewSpringDetector creates a new Spring detector.
func NewSpringDetector() *SpringDetector {
	return &SpringDetector{
		layerAnnotations: map[string]string{
			"Controller":       "controller",
			"RestController":   "controller",
			"Service":          "service",
			"Repository":       "repository",
			"Component":        "component",
			"Configuration":    "configuration",
			"Entity":           "domain",
			"Document":         "domain",
			"ControllerAdvice": "controller",
		},
		injectionAnnotations: map[string]bool{
			"Autowired": true,
			"Inject":    true,
			"Resource":  true,
			"Value":     true,
		},
		routeAnnotations: map[string]bool{
			"RequestMapping": true,
			"GetMapping":     true,
			"PostMapping":    true,
			"PutMapping":     true,
			"DeleteMapping":  true,
			"PatchMapping":   true,
		},
	}
}

// DetectLayer returns the architectural layer based on annotations.
func (d *SpringDetector) DetectLayer(annotations []string) string {
	for _, ann := range annotations {
		// Strip @prefix if present
		ann = strings.TrimPrefix(ann, "@")
		if layer, ok := d.layerAnnotations[ann]; ok {
			return layer
		}
	}
	return ""
}

// IsInjection returns true if any annotation indicates dependency injection.
func (d *SpringDetector) IsInjection(annotations []string) bool {
	for _, ann := range annotations {
		ann = strings.TrimPrefix(ann, "@")
		if d.injectionAnnotations[ann] {
			return true
		}
	}
	return false
}

// IsRouteAnnotation returns true if the annotation maps to an HTTP route.
func (d *SpringDetector) IsRouteAnnotation(annotation string) bool {
	annotation = strings.TrimPrefix(annotation, "@")
	return d.routeAnnotations[annotation]
}

// DetectRoute extracts the HTTP route from request mapping annotations.
func (d *SpringDetector) DetectRoute(annotations []string, node *sitter.Node, content []byte) string {
	for _, ann := range annotations {
		ann = strings.TrimPrefix(ann, "@")
		if d.routeAnnotations[ann] {
			// Try to extract route value from annotation
			route := d.extractRouteValue(node, content, ann)
			if route != "" {
				return route
			}
			// Return HTTP method as fallback
			return "/" + strings.ToLower(strings.TrimSuffix(ann, "Mapping"))
		}
	}
	return ""
}

// extractRouteValue extracts the path value from a mapping annotation.
func (d *SpringDetector) extractRouteValue(node *sitter.Node, content []byte, annName string) string {
	// Find the annotation node with the matching name
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "modifiers" {
			for j := 0; j < int(child.ChildCount()); j++ {
				mod := child.Child(j)
				if mod != nil && (mod.Type() == "annotation" || mod.Type() == "marker_annotation") {
					name := d.getAnnotationName(mod, content)
					if name == annName {
						return d.extractAnnotationValue(mod, content)
					}
				}
			}
		}
	}
	return ""
}

// getAnnotationName gets the name of an annotation node.
func (d *SpringDetector) getAnnotationName(node *sitter.Node, content []byte) string {
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		return nameNode.Content(content)
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "identifier" {
			return child.Content(content)
		}
	}
	return ""
}

// extractAnnotationValue extracts the value parameter from an annotation.
func (d *SpringDetector) extractAnnotationValue(node *sitter.Node, content []byte) string {
	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil {
		return ""
	}

	for i := 0; i < int(argsNode.ChildCount()); i++ {
		child := argsNode.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "string_literal":
			// Direct string value: @GetMapping("/path")
			return strings.Trim(child.Content(content), `"`)

		case "element_value_pair":
			// Named parameter: @RequestMapping(value = "/path")
			nameNode := child.ChildByFieldName("key")
			valueNode := child.ChildByFieldName("value")
			if nameNode != nil && valueNode != nil {
				keyName := nameNode.Content(content)
				if keyName == "value" || keyName == "path" {
					return strings.Trim(valueNode.Content(content), `"`)
				}
			}
		}
	}

	return ""
}

// DetectSpring checks if the file contains Spring annotations.
func (d *SpringDetector) DetectSpring(root *sitter.Node, content []byte) *provider.FrameworkInfo {
	var foundAnnotations []string
	var layer string

	d.walkForAnnotations(root, content, func(ann string) {
		ann = strings.TrimPrefix(ann, "@")
		foundAnnotations = append(foundAnnotations, ann)
		if l, ok := d.layerAnnotations[ann]; ok && layer == "" {
			layer = l
		}
	})

	// Check if any Spring annotations were found
	hasSpring := false
	for _, ann := range foundAnnotations {
		if d.layerAnnotations[ann] != "" || d.injectionAnnotations[ann] || d.routeAnnotations[ann] {
			hasSpring = true
			break
		}
	}

	if !hasSpring {
		return nil
	}

	return &provider.FrameworkInfo{
		Name:        "spring",
		Layer:       layer,
		Annotations: foundAnnotations,
	}
}

// walkForAnnotations walks the tree and collects annotations.
func (d *SpringDetector) walkForAnnotations(node *sitter.Node, content []byte, callback func(string)) {
	if node.Type() == "marker_annotation" || node.Type() == "annotation" {
		name := d.getAnnotationName(node, content)
		if name != "" {
			callback(name)
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			d.walkForAnnotations(child, content, callback)
		}
	}
}
