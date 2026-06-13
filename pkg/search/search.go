// Package search provides full-text search capabilities for knowledge graphs.
package search

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/plexusone/graphfs/pkg/graph"
)

// Searcher provides full-text search over graph data.
type Searcher struct {
	index     bleve.Index
	indexPath string
}

// SearchResult represents a single search hit.
type SearchResult struct {
	ID         string   `json:"id"`
	Score      float64  `json:"score"`
	Type       string   `json:"type"`
	Label      string   `json:"label,omitempty"`
	Package    string   `json:"package,omitempty"`
	SourceFile string   `json:"source_file,omitempty"`
	Snippet    string   `json:"snippet,omitempty"`
	Fragments  []string `json:"fragments,omitempty"`
}

// SearchOutput contains the full search response.
type SearchOutput struct {
	Query      string          `json:"query"`
	TotalHits  uint64          `json:"total_hits"`
	MaxScore   float64         `json:"max_score"`
	Results    []*SearchResult `json:"results"`
	Took       string          `json:"took"`
	Facets     map[string]any  `json:"facets,omitempty"`
	Truncated  bool            `json:"truncated,omitempty"`
	IndexStats *IndexStats     `json:"index_stats,omitempty"`
}

// IndexStats provides statistics about the search index.
type IndexStats struct {
	TotalDocs   uint64 `json:"total_docs"`
	IndexedAt   string `json:"indexed_at,omitempty"`
	IndexPath   string `json:"index_path,omitempty"`
	StorageSize int64  `json:"storage_size_bytes,omitempty"`
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	Limit     int      // Maximum results to return (default 20)
	Offset    int      // Skip first N results
	NodeTypes []string // Filter by node types (function, class, etc.)
	FuzzyDist int      // Fuzzy matching edit distance (0=exact, 1-2=fuzzy)
	Highlight bool     // Include highlighted fragments
}

// IndexedNode is the document type stored in bleve.
type IndexedNode struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Label      string `json:"label"`
	Doc        string `json:"doc"`         // Docstring/description
	Package    string `json:"package"`     // Package name
	SourceFile string `json:"source_file"` // File path
	Module     string `json:"module"`      // Module name
	Signature  string `json:"signature"`   // Function/method signature
	AllText    string `json:"all_text"`    // Combined searchable text
}

// NewSearcher creates a new searcher, opening or creating the index.
func NewSearcher(graphPath string) (*Searcher, error) {
	indexPath := filepath.Join(graphPath, "search_index")

	// Try to open existing index
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Create new index with mapping
		indexMapping := buildIndexMapping()
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("creating index: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("opening index: %w", err)
	}

	return &Searcher{
		index:     index,
		indexPath: indexPath,
	}, nil
}

// buildIndexMapping creates the bleve index mapping for nodes.
func buildIndexMapping() mapping.IndexMapping {
	// Node document mapping
	nodeMapping := bleve.NewDocumentMapping()

	// Keyword fields (exact match)
	keywordField := bleve.NewTextFieldMapping()
	keywordField.Analyzer = keyword.Name

	nodeMapping.AddFieldMappingsAt("id", keywordField)
	nodeMapping.AddFieldMappingsAt("type", keywordField)
	nodeMapping.AddFieldMappingsAt("package", keywordField)

	// Standard text fields (tokenized, stemmed)
	textField := bleve.NewTextFieldMapping()
	textField.Analyzer = standard.Name
	textField.Store = true
	textField.IncludeTermVectors = true

	nodeMapping.AddFieldMappingsAt("label", textField)
	nodeMapping.AddFieldMappingsAt("doc", textField)
	nodeMapping.AddFieldMappingsAt("source_file", textField)
	nodeMapping.AddFieldMappingsAt("signature", textField)
	nodeMapping.AddFieldMappingsAt("all_text", textField)

	// Index mapping
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = nodeMapping
	indexMapping.DefaultAnalyzer = standard.Name

	return indexMapping
}

// IndexNodes adds nodes to the search index.
func (s *Searcher) IndexNodes(nodes []*graph.Node) error {
	batch := s.index.NewBatch()

	for _, n := range nodes {
		doc := nodeToIndexed(n)
		if err := batch.Index(n.ID, doc); err != nil {
			return fmt.Errorf("indexing node %s: %w", n.ID, err)
		}
	}

	if err := s.index.Batch(batch); err != nil {
		return fmt.Errorf("committing batch: %w", err)
	}

	return nil
}

// nodeToIndexed converts a graph node to an indexed document.
func nodeToIndexed(n *graph.Node) *IndexedNode {
	doc := &IndexedNode{
		ID:    n.ID,
		Type:  n.Type,
		Label: n.Label,
	}

	if n.Attrs != nil {
		doc.Doc = n.Attrs["doc"]
		doc.Package = n.Attrs["package"]
		doc.SourceFile = n.Attrs["source_file"]
		doc.Module = n.Attrs["module"]
		doc.Signature = n.Attrs["signature"]
	}

	// Build combined text for broad matching
	doc.AllText = fmt.Sprintf("%s %s %s %s %s",
		n.Label, doc.Doc, doc.Package, doc.Signature, n.ID)

	return doc
}

// Search performs a full-text search.
func (s *Searcher) Search(queryStr string, opts SearchOptions) (*SearchOutput, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	// Build query
	var searchQuery query.Query
	if opts.FuzzyDist > 0 {
		fuzzyQuery := bleve.NewFuzzyQuery(queryStr)
		fuzzyQuery.Fuzziness = opts.FuzzyDist
		searchQuery = fuzzyQuery
	} else {
		// Use match query for natural language search
		searchQuery = bleve.NewMatchQuery(queryStr)
	}

	// Apply type filter if specified
	if len(opts.NodeTypes) > 0 {
		typeQueries := make([]query.Query, len(opts.NodeTypes))
		for i, t := range opts.NodeTypes {
			termQuery := bleve.NewTermQuery(t)
			termQuery.SetField("type")
			typeQueries[i] = termQuery
		}
		typeFilter := bleve.NewDisjunctionQuery(typeQueries...)
		searchQuery = bleve.NewConjunctionQuery(searchQuery, typeFilter)
	}

	// Build search request
	searchReq := bleve.NewSearchRequestOptions(searchQuery, opts.Limit, opts.Offset, false)
	searchReq.Fields = []string{"id", "type", "label", "package", "source_file", "doc"}

	// Add highlighting if requested
	if opts.Highlight {
		searchReq.Highlight = bleve.NewHighlight()
	}

	// Add type facet
	searchReq.AddFacet("types", bleve.NewFacetRequest("type", 10))

	// Execute search
	searchRes, err := s.index.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}

	// Convert results
	output := &SearchOutput{
		Query:     queryStr,
		TotalHits: searchRes.Total,
		MaxScore:  searchRes.MaxScore,
		Took:      searchRes.Took.String(),
		Results:   make([]*SearchResult, 0, len(searchRes.Hits)),
	}

	for _, hit := range searchRes.Hits {
		result := &SearchResult{
			ID:    hit.ID,
			Score: hit.Score,
		}

		// Extract fields
		if v, ok := hit.Fields["type"].(string); ok {
			result.Type = v
		}
		if v, ok := hit.Fields["label"].(string); ok {
			result.Label = v
		}
		if v, ok := hit.Fields["package"].(string); ok {
			result.Package = v
		}
		if v, ok := hit.Fields["source_file"].(string); ok {
			result.SourceFile = v
		}
		if v, ok := hit.Fields["doc"].(string); ok && len(v) > 0 {
			// Truncate snippet
			if len(v) > 150 {
				result.Snippet = v[:150] + "..."
			} else {
				result.Snippet = v
			}
		}

		// Add highlighted fragments
		if opts.Highlight && len(hit.Fragments) > 0 {
			for _, frags := range hit.Fragments {
				result.Fragments = append(result.Fragments, frags...)
			}
		}

		output.Results = append(output.Results, result)
	}

	// Add facets
	if searchRes.Facets != nil {
		output.Facets = make(map[string]any)
		for name, facet := range searchRes.Facets {
			terms := make([]map[string]any, 0, len(facet.Terms.Terms()))
			for _, term := range facet.Terms.Terms() {
				terms = append(terms, map[string]any{
					"term":  term.Term,
					"count": term.Count,
				})
			}
			output.Facets[name] = terms
		}
	}

	if searchRes.Total > uint64(opts.Limit+opts.Offset) { //nolint:gosec // Safe: opts are validated non-negative
		output.Truncated = true
	}

	return output, nil
}

// Stats returns index statistics.
func (s *Searcher) Stats() (*IndexStats, error) {
	docCount, err := s.index.DocCount()
	if err != nil {
		return nil, fmt.Errorf("getting doc count: %w", err)
	}

	stats := &IndexStats{
		TotalDocs: docCount,
		IndexPath: s.indexPath,
	}

	// Try to get storage size
	var totalSize int64
	err = filepath.Walk(s.indexPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err == nil {
		stats.StorageSize = totalSize
	}

	return stats, nil
}

// Close closes the search index.
func (s *Searcher) Close() error {
	return s.index.Close()
}

// Reindex clears and rebuilds the index from nodes.
func (s *Searcher) Reindex(nodes []*graph.Node) error {
	// Delete all existing documents
	docCount, _ := s.index.DocCount()
	if docCount > 0 {
		// Get all doc IDs
		query := bleve.NewMatchAllQuery()
		searchReq := bleve.NewSearchRequest(query)
		searchReq.Size = int(docCount) //nolint:gosec // Safe: docCount limited by index size
		searchReq.Fields = []string{}

		results, err := s.index.Search(searchReq)
		if err != nil {
			return fmt.Errorf("listing existing docs: %w", err)
		}

		batch := s.index.NewBatch()
		for _, hit := range results.Hits {
			batch.Delete(hit.ID)
		}
		if err := s.index.Batch(batch); err != nil {
			return fmt.Errorf("deleting existing docs: %w", err)
		}
	}

	// Index new nodes
	return s.IndexNodes(nodes)
}
