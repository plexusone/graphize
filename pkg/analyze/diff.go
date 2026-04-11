// Package analyze provides graph analysis functions.
package analyze

import "github.com/plexusone/graphfs/pkg/analyze"

// Re-export diff types from graphfs for backward compatibility
type (
	GraphDiff  = analyze.GraphDiff
	NodeChange = analyze.NodeChange
	EdgeChange = analyze.EdgeChange
)

// Re-export diff functions
var DiffGraphs = analyze.DiffGraphs
