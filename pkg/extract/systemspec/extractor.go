// Package systemspec provides system-spec extraction for knowledge graphs.
// It extracts infrastructure topology from system-spec JSON files, enabling
// queries that span both code and infrastructure.
package systemspec

import (
	"github.com/plexusone/graphfs/pkg/graph"
	"github.com/plexusone/graphize/provider"
	sysspec "github.com/plexusone/system-spec/graphize"
)

const (
	// Language is the canonical name for system-spec.
	Language = "system-spec"

	// NodePrefix is the prefix for system-spec node IDs.
	// Note: system-spec uses its own prefixes (svc:, rds:, etc.) so this is not used.
	NodePrefix = ""
)

// Extractor implements provider.LanguageExtractor for system-spec JSON files.
// It wraps the system-spec graphize.Provider to integrate with graphize's
// extraction pipeline.
type Extractor struct {
	provider *sysspec.Provider
}

// New creates a new system-spec extractor.
func New() *Extractor {
	return &Extractor{
		provider: sysspec.NewProvider(),
	}
}

// Language returns "system-spec".
func (e *Extractor) Language() string {
	return Language
}

// Extensions returns JSON file extension.
// Note: Not all JSON files are system-spec files; CanExtract does content detection.
func (e *Extractor) Extensions() []string {
	return e.provider.Extensions()
}

// CanExtract returns true if the file is a system-spec JSON document.
// It checks for the presence of "name" and "services" fields at the root.
func (e *Extractor) CanExtract(path string) bool {
	return e.provider.CanExtract(path)
}

// ExtractFile extracts nodes and edges from a system-spec JSON file.
// It produces:
//   - System node (system:<name>)
//   - Service nodes (svc:<name>) with links_to edges to repos
//   - Cloud resource nodes (rds:, sqs:, s3:, etc.)
//   - Connection edges between services
//   - Deployment nodes (helm:, terraform:)
func (e *Extractor) ExtractFile(path, baseDir string) ([]*graph.Node, []*graph.Edge, error) {
	return e.provider.ExtractFile(path, baseDir)
}

// DetectFramework returns nil as system-spec is not a code framework.
func (e *Extractor) DetectFramework(path string) *provider.FrameworkInfo {
	return nil
}

func init() {
	provider.Register(func() provider.LanguageExtractor {
		return New()
	}, provider.PriorityDefault)
}
