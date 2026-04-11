// Package source manages multi-repo source tracking with git commit awareness.
package source

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Source represents a tracked source repository.
type Source struct {
	// Path is the absolute path to the repository root.
	Path string `json:"path"`

	// Commit is the git commit hash that was last analyzed.
	Commit string `json:"commit"`

	// Branch is the git branch that was checked out during analysis.
	Branch string `json:"branch"`

	// AnalyzedAt is when the source was last analyzed.
	AnalyzedAt time.Time `json:"analyzed_at"`
}

// Manifest tracks all sources in a Graphize database.
type Manifest struct {
	// Sources is the list of tracked repositories.
	Sources []*Source `json:"sources"`

	// CreatedAt is when the manifest was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the manifest was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// NewManifest creates an empty manifest.
func NewManifest() *Manifest {
	now := time.Now().UTC()
	return &Manifest{
		Sources:   make([]*Source, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddSource adds or updates a source in the manifest.
func (m *Manifest) AddSource(s *Source) {
	// Check if source already exists
	for i, existing := range m.Sources {
		if existing.Path == s.Path {
			m.Sources[i] = s
			m.UpdatedAt = time.Now().UTC()
			return
		}
	}
	m.Sources = append(m.Sources, s)
	m.UpdatedAt = time.Now().UTC()
}

// GetSource returns a source by path, or nil if not found.
func (m *Manifest) GetSource(path string) *Source {
	for _, s := range m.Sources {
		if s.Path == path {
			return s
		}
	}
	return nil
}

// SourceStatus represents the currency status of a source.
type SourceStatus struct {
	Source        *Source
	CurrentCommit string
	CurrentBranch string
	IsStale       bool
	CommitsBehind int // -1 if unknown
}

// CheckStatus checks if a source is current with git HEAD.
func CheckStatus(s *Source) (*SourceStatus, error) {
	currentCommit, err := getGitCommit(s.Path)
	if err != nil {
		return nil, fmt.Errorf("getting current commit: %w", err)
	}

	currentBranch, err := getGitBranch(s.Path)
	if err != nil {
		return nil, fmt.Errorf("getting current branch: %w", err)
	}

	status := &SourceStatus{
		Source:        s,
		CurrentCommit: currentCommit,
		CurrentBranch: currentBranch,
		IsStale:       currentCommit != s.Commit,
		CommitsBehind: -1,
	}

	// Try to count commits behind
	if status.IsStale {
		behind, err := countCommitsBehind(s.Path, s.Commit, currentCommit)
		if err == nil {
			status.CommitsBehind = behind
		}
	} else {
		status.CommitsBehind = 0
	}

	return status, nil
}

// NewSourceFromPath creates a Source from a git repository path.
func NewSourceFromPath(path string) (*Source, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	commit, err := getGitCommit(absPath)
	if err != nil {
		return nil, fmt.Errorf("getting git commit: %w", err)
	}

	branch, err := getGitBranch(absPath)
	if err != nil {
		return nil, fmt.Errorf("getting git branch: %w", err)
	}

	return &Source{
		Path:       absPath,
		Commit:     commit,
		Branch:     branch,
		AnalyzedAt: time.Now().UTC(),
	}, nil
}

// getGitCommit returns the current HEAD commit hash.
func getGitCommit(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// getGitBranch returns the current branch name.
func getGitBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// countCommitsBehind counts commits between old and new.
func countCommitsBehind(repoPath, oldCommit, newCommit string) (int, error) {
	cmd := exec.Command("git", "rev-list", "--count", oldCommit+".."+newCommit)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return -1, err
	}
	var count int
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count)
	return count, err
}
