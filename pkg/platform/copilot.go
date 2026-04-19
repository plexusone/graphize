package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// CopilotInstaller integrates graphize with GitHub Copilot.
type CopilotInstaller struct{}

func init() {
	Register(&CopilotInstaller{})
}

// Name returns "copilot".
func (c *CopilotInstaller) Name() string {
	return "copilot"
}

// Description returns the platform description.
func (c *CopilotInstaller) Description() string {
	return "GitHub Copilot skills integration"
}

// Install adds graphize skill to GitHub Copilot configuration.
func (c *CopilotInstaller) Install(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	skillsDir := filepath.Join(projectPath, ".github", "copilot", "skills")
	skillFile := filepath.Join(skillsDir, "graphize.md")

	// Check if already installed
	if _, err := os.Stat(skillFile); err == nil && !opts.Force {
		return fmt.Errorf("graphize skill already exists at %s (use --force to overwrite)", skillFile)
	}

	// Ensure directory exists
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("creating skills directory: %w", err)
	}

	graphPath := opts.GraphPath
	if graphPath == "" {
		graphPath = ".graphize"
	}

	content := generateCopilotSkill(graphPath)

	if opts.DryRun {
		fmt.Printf("Would write to %s:\n%s\n", skillFile, content)
		return nil
	}

	if err := os.WriteFile(skillFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing skill file: %w", err)
	}

	return nil
}

// Uninstall removes graphize skill from GitHub Copilot configuration.
func (c *CopilotInstaller) Uninstall(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	skillFile := filepath.Join(projectPath, ".github", "copilot", "skills", "graphize.md")

	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		return fmt.Errorf("graphize skill not found")
	}

	if opts.DryRun {
		fmt.Printf("Would remove %s\n", skillFile)
		return nil
	}

	if err := os.Remove(skillFile); err != nil {
		return fmt.Errorf("removing skill file: %w", err)
	}

	return nil
}

// Status checks if graphize skill is configured for GitHub Copilot.
func (c *CopilotInstaller) Status(opts InstallOptions) (*Status, error) {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	skillFile := filepath.Join(projectPath, ".github", "copilot", "skills", "graphize.md")

	status := &Status{
		ConfigPath: skillFile,
		Details:    make(map[string]string),
	}

	info, err := os.Stat(skillFile)
	if os.IsNotExist(err) {
		status.Message = "graphize skill not configured"
		return status, nil
	} else if err != nil {
		return nil, fmt.Errorf("checking skill file: %w", err)
	}

	status.Installed = true
	status.Message = "graphize skill configured"
	status.Details["size"] = fmt.Sprintf("%d bytes", info.Size())
	status.Details["modified"] = info.ModTime().Format("2006-01-02 15:04:05")

	return status, nil
}

// generateCopilotSkill creates the content for the GitHub Copilot skill.
func generateCopilotSkill(graphPath string) string {
	return fmt.Sprintf(`# Graphize Knowledge Graph Skill

Use graphize to query the codebase knowledge graph.

## Commands

### Query the Graph

`+"`"+``+"`"+``+"`"+`bash
graphize query <node-id>           # Show edges for a node
graphize query <node-id> --depth 2 # Traverse 2 levels deep
graphize query --type calls        # Filter by edge type
`+"`"+``+"`"+``+"`"+`

### Explain a Node

`+"`"+``+"`"+``+"`"+`bash
graphize explain <node-id>         # Get full context about a node
graphize explain <node-id> --json  # Output as JSON
`+"`"+``+"`"+``+"`"+`

### Generate Report

`+"`"+``+"`"+``+"`"+`bash
graphize report                    # Full analysis report
graphize report --health           # Corpus health metrics
graphize report --top 5            # Limit results
`+"`"+``+"`"+``+"`"+`

## Node ID Patterns

- Functions: `+"`"+`func_filename.go.FunctionName`+"`"+`
- Methods: `+"`"+`method_ReceiverType.MethodName`+"`"+`
- Types: `+"`"+`type_TypeName`+"`"+`
- Packages: `+"`"+`pkg_packagename`+"`"+`
- Files: `+"`"+`file_path/to/file.go`+"`"+`

## Graph Location

The knowledge graph is at: %s

## Use Cases

1. **Find dependencies**: `+"`"+`graphize query type_Server --depth 3`+"`"+`
2. **Understand architecture**: `+"`"+`graphize report`+"`"+`
3. **Explore relationships**: `+"`"+`graphize explain func_main`+"`"+`
4. **Find surprising connections**: `+"`"+`graphize report --top 10`+"`"+`
`, graphPath)
}
