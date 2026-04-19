package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AiderInstaller integrates graphize with Aider.
type AiderInstaller struct{}

func init() {
	Register(&AiderInstaller{})
}

// Name returns "aider".
func (a *AiderInstaller) Name() string {
	return "aider"
}

// Description returns the platform description.
func (a *AiderInstaller) Description() string {
	return "Aider AGENTS.md integration"
}

// Install adds graphize section to AGENTS.md.
func (a *AiderInstaller) Install(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	agentsFile := filepath.Join(projectPath, "AGENTS.md")

	// Read existing content or start fresh
	var existingContent string
	if data, err := os.ReadFile(agentsFile); err == nil { //nolint:gosec // G304: path is from known location
		existingContent = string(data)
	}

	// Check if graphize section already exists
	if strings.Contains(existingContent, "## Graphize Knowledge Graph") {
		if !opts.Force {
			return fmt.Errorf("graphize section already exists in AGENTS.md (use --force to overwrite)")
		}
		// Remove existing section
		existingContent = removeGraphizeSection(existingContent)
	}

	graphPath := opts.GraphPath
	if graphPath == "" {
		graphPath = ".graphize"
	}

	section := generateAiderSection(graphPath)

	// Append or create
	var newContent string
	if existingContent == "" {
		newContent = "# AGENTS.md\n\n" + section
	} else {
		newContent = strings.TrimRight(existingContent, "\n") + "\n\n" + section
	}

	if opts.DryRun {
		fmt.Printf("Would write to %s:\n%s\n", agentsFile, newContent)
		return nil
	}

	if err := os.WriteFile(agentsFile, []byte(newContent), 0600); err != nil { //nolint:gosec // G703: path is from controlled project directory
		return fmt.Errorf("writing AGENTS.md: %w", err)
	}

	return nil
}

// Uninstall removes graphize section from AGENTS.md.
func (a *AiderInstaller) Uninstall(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	agentsFile := filepath.Join(projectPath, "AGENTS.md")

	data, err := os.ReadFile(agentsFile) //nolint:gosec // G304: path is from known location
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("AGENTS.md not found")
		}
		return fmt.Errorf("reading AGENTS.md: %w", err)
	}

	content := string(data)
	if !strings.Contains(content, "## Graphize Knowledge Graph") {
		return fmt.Errorf("graphize section not found in AGENTS.md")
	}

	newContent := removeGraphizeSection(content)

	if opts.DryRun {
		fmt.Printf("Would write to %s:\n%s\n", agentsFile, newContent)
		return nil
	}

	if err := os.WriteFile(agentsFile, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("writing AGENTS.md: %w", err)
	}

	return nil
}

// Status checks if graphize section exists in AGENTS.md.
func (a *AiderInstaller) Status(opts InstallOptions) (*Status, error) {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	agentsFile := filepath.Join(projectPath, "AGENTS.md")

	status := &Status{
		ConfigPath: agentsFile,
		Details:    make(map[string]string),
	}

	data, err := os.ReadFile(agentsFile) //nolint:gosec // G304: path is from known location
	if os.IsNotExist(err) {
		status.Message = "AGENTS.md not found"
		return status, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading AGENTS.md: %w", err)
	}

	content := string(data)
	if !strings.Contains(content, "## Graphize Knowledge Graph") {
		status.Message = "graphize section not found"
		return status, nil
	}

	status.Installed = true
	status.Message = "graphize section configured"

	return status, nil
}

// removeGraphizeSection removes the graphize section from AGENTS.md content.
func removeGraphizeSection(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## Graphize Knowledge Graph") {
			inSection = true
			continue
		}

		// End section on next h2 heading
		if inSection && strings.HasPrefix(line, "## ") {
			inSection = false
		}

		if !inSection {
			result = append(result, line)
		}
	}

	return strings.TrimRight(strings.Join(result, "\n"), "\n") + "\n"
}

// generateAiderSection creates the graphize section for AGENTS.md.
func generateAiderSection(graphPath string) string {
	return fmt.Sprintf(`## Graphize Knowledge Graph

This project uses graphize for knowledge graph-based code navigation.

### Commands

Query the knowledge graph:

`+"`"+``+"`"+``+"`"+`bash
graphize query <node-id>           # Show edges for a node
graphize query <node-id> --depth 2 # Traverse 2 levels deep
graphize explain <node-id>         # Get full context
graphize report                    # Architecture report
graphize report --health           # Corpus health
`+"`"+``+"`"+``+"`"+`

### Node ID Patterns

| Pattern | Example |
|---------|---------|
| Functions | `+"`"+`func_main.go.HandleRequest`+"`"+` |
| Methods | `+"`"+`method_Server.ServeHTTP`+"`"+` |
| Types | `+"`"+`type_Config`+"`"+` |
| Packages | `+"`"+`pkg_api`+"`"+` |

### Graph Location

%s

### When to Use

1. **Understanding dependencies**: Before modifying code, query its connections
2. **Finding entry points**: Use report to find god nodes
3. **Exploring architecture**: Use explain for detailed node context
`, graphPath)
}
