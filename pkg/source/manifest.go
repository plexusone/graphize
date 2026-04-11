package source

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const manifestFilename = "manifest.json"

// LoadManifest loads the manifest from a graph directory.
// Returns an empty manifest if the file doesn't exist.
func LoadManifest(graphPath string) (*Manifest, error) {
	path := filepath.Join(graphPath, manifestFilename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewManifest(), nil
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &m, nil
}

// SaveManifest saves the manifest to a graph directory.
func SaveManifest(graphPath string, m *Manifest) error {
	path := filepath.Join(graphPath, manifestFilename)

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}

// RemoveSource removes a source from the manifest by path.
func (m *Manifest) RemoveSource(path string) bool {
	for i, s := range m.Sources {
		if s.Path == path {
			m.Sources = append(m.Sources[:i], m.Sources[i+1:]...)
			return true
		}
	}
	return false
}

// CheckAllStatus checks the status of all sources in the manifest.
func (m *Manifest) CheckAllStatus() ([]*SourceStatus, error) {
	var statuses []*SourceStatus
	for _, s := range m.Sources {
		status, err := CheckStatus(s)
		if err != nil {
			// Include error in status rather than failing completely
			statuses = append(statuses, &SourceStatus{
				Source:        s,
				IsStale:       true,
				CommitsBehind: -1,
			})
			continue
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}
