// Package markdown provides Markdown/text extraction for knowledge graphs.
// It extracts documentation concepts that can be linked to code entities.
package markdown

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/provider"
)

const (
	// Language is the canonical name for Markdown.
	Language = "markdown"

	// NodePrefix is the prefix for all Markdown node IDs.
	NodePrefix = "doc_"
)

// Semantic edge types for documentation relationships.
const (
	EdgeTypeDocuments = "documents" // Doc concept documents code entity
	EdgeTypeDescribes = "describes" // Doc describes behavior/architecture
	EdgeTypeExplains  = "explains"  // Doc provides design rationale
)

// Extractor implements provider.LanguageExtractor for Markdown files.
type Extractor struct{}

// New creates a new Markdown extractor.
func New() *Extractor {
	return &Extractor{}
}

// Language returns "markdown".
func (e *Extractor) Language() string {
	return Language
}

// Extensions returns Markdown file extensions.
func (e *Extractor) Extensions() []string {
	return []string{".md", ".markdown", ".txt", ".rst"}
}

// CanExtract returns true for Markdown/text files.
func (e *Extractor) CanExtract(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supported := range e.Extensions() {
		if ext == supported {
			return true
		}
	}
	return false
}

// ExtractFile extracts nodes and edges from a Markdown file.
func (e *Extractor) ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error) {
	content, err := os.ReadFile(path) //nolint:gosec // G304: path comes from trusted source
	if err != nil {
		return nil, nil, err
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Get relative path for cleaner IDs
	relPath, _ := filepath.Rel(baseDir, path)
	if relPath == "" {
		relPath = path
	}

	// Create file node
	fileID := makeID("file", relPath)
	fileLabel := filepath.Base(path)
	nodes = append(nodes, &graph.Node{
		ID:    fileID,
		Type:  graph.NodeTypeFile,
		Label: fileLabel,
		Attrs: map[string]string{
			"path":     relPath,
			"language": Language,
			"doc_type": classifyDocType(path),
		},
	})

	// Parse content
	concepts := parseMarkdown(string(content))

	// Create concept nodes and edges
	for _, concept := range concepts {
		conceptID := makeID("concept", relPath+"_"+sanitizeID(concept.Title))
		nodes = append(nodes, &graph.Node{
			ID:    conceptID,
			Type:  "concept",
			Label: concept.Title,
			Attrs: map[string]string{
				"source_file": relPath,
				"level":       itoa(concept.Level),
				"language":    Language,
			},
		})

		// Edge from concept to file
		edges = append(edges, &graph.Edge{
			From:       fileID,
			To:         conceptID,
			Type:       graph.EdgeTypeContains,
			Confidence: graph.ConfidenceExtracted,
		})

		// Create edges for code references found in the content
		for _, ref := range concept.CodeRefs {
			edges = append(edges, &graph.Edge{
				From:            conceptID,
				To:              ref.Target,
				Type:            ref.EdgeType,
				Confidence:      graph.ConfidenceInferred,
				ConfidenceScore: 0.7,
				Attrs: map[string]string{
					"reason": ref.Reason,
					"source": "doc_extraction",
				},
			})
		}
	}

	return nodes, edges, nil
}

// DetectFramework returns nil for Markdown files (no framework detection).
func (e *Extractor) DetectFramework(path string) *provider.FrameworkInfo {
	return nil
}

// Concept represents a documentation concept extracted from Markdown.
type Concept struct {
	Title    string
	Level    int    // Heading level (1-6)
	Content  string // Text content under this heading
	CodeRefs []CodeRef
}

// CodeRef represents a reference to code from documentation.
type CodeRef struct {
	Target   string // Target node ID
	EdgeType string // documents, describes, explains
	Reason   string // Why this reference exists
}

// parseMarkdown extracts concepts from Markdown content.
func parseMarkdown(content string) []Concept {
	var concepts []Concept
	lines := strings.Split(content, "\n")

	var currentConcept *Concept
	var contentLines []string

	headingPattern := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	codeBlockPattern := regexp.MustCompile("^```(\\w+)?")
	inCodeBlock := false
	codeBlockLang := ""

	for _, line := range lines {
		// Track code blocks
		if matches := codeBlockPattern.FindStringSubmatch(line); len(matches) > 0 {
			if !inCodeBlock {
				inCodeBlock = true
				if len(matches) > 1 {
					codeBlockLang = matches[1]
				}
			} else {
				// Closing code block - extract code references
				if currentConcept != nil && codeBlockLang != "" {
					currentConcept.CodeRefs = append(currentConcept.CodeRefs,
						extractCodeRefsFromBlock(strings.Join(contentLines, "\n"), codeBlockLang)...)
				}
				inCodeBlock = false
				codeBlockLang = ""
			}
			continue
		}

		if inCodeBlock {
			contentLines = append(contentLines, line)
			continue
		}

		// Check for headings
		if matches := headingPattern.FindStringSubmatch(line); len(matches) > 0 {
			// Save previous concept
			if currentConcept != nil {
				currentConcept.Content = strings.Join(contentLines, "\n")
				// Extract inline code references
				currentConcept.CodeRefs = append(currentConcept.CodeRefs,
					extractInlineCodeRefs(currentConcept.Content)...)
				concepts = append(concepts, *currentConcept)
			}

			level := len(matches[1])
			title := strings.TrimSpace(matches[2])

			currentConcept = &Concept{
				Title: title,
				Level: level,
			}
			contentLines = nil
		} else {
			contentLines = append(contentLines, line)
		}
	}

	// Save last concept
	if currentConcept != nil {
		currentConcept.Content = strings.Join(contentLines, "\n")
		currentConcept.CodeRefs = append(currentConcept.CodeRefs,
			extractInlineCodeRefs(currentConcept.Content)...)
		concepts = append(concepts, *currentConcept)
	}

	return concepts
}

// extractInlineCodeRefs extracts code references from inline code.
func extractInlineCodeRefs(content string) []CodeRef {
	var refs []CodeRef

	// Match inline code like `FunctionName`, `TypeName`, `package.Function`
	inlineCodePattern := regexp.MustCompile("`([A-Z][a-zA-Z0-9]*(?:\\.[A-Z][a-zA-Z0-9]*)?)`")
	matches := inlineCodePattern.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			ref := match[1]
			if seen[ref] {
				continue
			}
			seen[ref] = true

			// Determine node ID pattern
			nodeID := ""
			edgeType := EdgeTypeDocuments

			if strings.Contains(ref, ".") {
				// Method reference: Type.Method
				parts := strings.Split(ref, ".")
				nodeID = "method_" + parts[0] + "." + parts[1]
			} else if isCapitalized(ref) {
				// Could be type or function
				// Default to type for capitalized names
				nodeID = "type_" + ref
			}

			if nodeID != "" {
				refs = append(refs, CodeRef{
					Target:   nodeID,
					EdgeType: edgeType,
					Reason:   "Referenced in documentation",
				})
			}
		}
	}

	return refs
}

// extractCodeRefsFromBlock extracts references from code blocks.
func extractCodeRefsFromBlock(content, lang string) []CodeRef {
	var refs []CodeRef

	// For Go code blocks, extract function and type names
	if lang == "go" || lang == "golang" {
		funcPattern := regexp.MustCompile(`\bfunc\s+(\w+)\s*\(`)
		typePattern := regexp.MustCompile(`\btype\s+(\w+)\s+`)

		for _, match := range funcPattern.FindAllStringSubmatch(content, -1) {
			if len(match) > 1 {
				refs = append(refs, CodeRef{
					Target:   "func_" + match[1],
					EdgeType: EdgeTypeDescribes,
					Reason:   "Go function shown in code block",
				})
			}
		}

		for _, match := range typePattern.FindAllStringSubmatch(content, -1) {
			if len(match) > 1 {
				refs = append(refs, CodeRef{
					Target:   "type_" + match[1],
					EdgeType: EdgeTypeDescribes,
					Reason:   "Go type shown in code block",
				})
			}
		}
	}

	return refs
}

// classifyDocType determines the type of documentation file.
func classifyDocType(path string) string {
	base := strings.ToLower(filepath.Base(path))
	dir := strings.ToLower(filepath.Dir(path))

	switch {
	case base == "readme.md" || base == "readme":
		return "readme"
	case base == "changelog.md" || base == "changelog":
		return "changelog"
	case base == "contributing.md":
		return "contributing"
	case base == "license" || base == "license.md":
		return "license"
	case strings.Contains(dir, "docs") || strings.Contains(dir, "documentation"):
		return "documentation"
	case strings.Contains(base, "spec") || strings.Contains(base, "rfc"):
		return "specification"
	case strings.Contains(base, "design") || strings.Contains(base, "architecture"):
		return "architecture"
	case strings.Contains(base, "api"):
		return "api_reference"
	default:
		return "general"
	}
}

// Helper functions

func makeID(prefix, name string) string {
	return NodePrefix + prefix + "_" + sanitizeID(name)
}

func sanitizeID(s string) string {
	// Replace path separators and special chars
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return strings.ToLower(s)
}

func isCapitalized(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= 'A' && s[0] <= 'Z'
}

func itoa(i int) string {
	digits := "0123456789"
	if i >= 0 && i <= 9 {
		return string(digits[i])
	}
	return fmt.Sprintf("%d", i)
}

// CollectDocFiles walks a directory and returns all documentation files.
func CollectDocFiles(dir string, skipDirs []string) ([]string, error) {
	var files []string
	extractor := New()

	skipSet := make(map[string]bool)
	for _, d := range skipDirs {
		skipSet[d] = true
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			// Skip hidden and vendor directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || skipSet[name] {
				return filepath.SkipDir
			}
			return nil
		}

		if extractor.CanExtract(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ReadDocFiles reads documentation files and returns their paths and estimated tokens.
func ReadDocFiles(dir string) ([]DocFile, error) {
	files, err := CollectDocFiles(dir, []string{"vendor", "node_modules", "testdata"})
	if err != nil {
		return nil, err
	}

	var docs []DocFile
	for _, path := range files {
		content, err := os.ReadFile(path) //nolint:gosec // G304: path from trusted source
		if err != nil {
			continue
		}

		// Rough token estimate
		words := len(strings.Fields(string(content)))
		tokens := int(float64(words) * 1.3)

		docs = append(docs, DocFile{
			Path:   path,
			Tokens: tokens,
			Type:   classifyDocType(path),
		})
	}

	return docs, nil
}

// DocFile represents a documentation file with metadata.
type DocFile struct {
	Path   string `json:"path"`
	Tokens int    `json:"tokens"`
	Type   string `json:"type"`
}

// init registers the Markdown extractor with the provider registry.
func init() {
	provider.Register(func() provider.LanguageExtractor {
		return New()
	}, provider.PriorityDefault)
}

// ScanDocFile reads a doc file and returns a line scanner.
func ScanDocFile(path string) (*bufio.Scanner, *os.File, error) {
	f, err := os.Open(path) //nolint:gosec // G304: path from trusted source
	if err != nil {
		return nil, nil, err
	}
	return bufio.NewScanner(f), f, nil
}
