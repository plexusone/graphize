package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// CursorInstaller integrates graphize with Cursor IDE.
type CursorInstaller struct{}

func init() {
	Register(&CursorInstaller{})
}

// Name returns "cursor".
func (c *CursorInstaller) Name() string {
	return "cursor"
}

// Description returns the platform description.
func (c *CursorInstaller) Description() string {
	return "Cursor IDE rules file integration"
}

// Install adds graphize rules to Cursor configuration.
func (c *CursorInstaller) Install(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	rulesDir := filepath.Join(projectPath, ".cursor", "rules")
	rulesFile := filepath.Join(rulesDir, "graphize.mdc")

	// Check if already installed
	if _, err := os.Stat(rulesFile); err == nil && !opts.Force {
		return fmt.Errorf("graphize rules already exist at %s (use --force to overwrite)", rulesFile)
	}

	// Ensure directory exists
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("creating rules directory: %w", err)
	}

	graphPath := opts.GraphPath
	if graphPath == "" {
		graphPath = ".graphize"
	}

	content := generateCursorRules(graphPath)

	if opts.DryRun {
		fmt.Printf("Would write to %s:\n%s\n", rulesFile, content)
		return nil
	}

	if err := os.WriteFile(rulesFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing rules file: %w", err)
	}

	return nil
}

// Uninstall removes graphize rules from Cursor configuration.
func (c *CursorInstaller) Uninstall(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	rulesFile := filepath.Join(projectPath, ".cursor", "rules", "graphize.mdc")

	if _, err := os.Stat(rulesFile); os.IsNotExist(err) {
		return fmt.Errorf("graphize rules not found")
	}

	if opts.DryRun {
		fmt.Printf("Would remove %s\n", rulesFile)
		return nil
	}

	if err := os.Remove(rulesFile); err != nil {
		return fmt.Errorf("removing rules file: %w", err)
	}

	return nil
}

// Status checks if graphize rules are configured for Cursor.
func (c *CursorInstaller) Status(opts InstallOptions) (*Status, error) {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	rulesFile := filepath.Join(projectPath, ".cursor", "rules", "graphize.mdc")

	status := &Status{
		ConfigPath: rulesFile,
		Details:    make(map[string]string),
	}

	info, err := os.Stat(rulesFile)
	if os.IsNotExist(err) {
		status.Message = "graphize rules not configured"
		return status, nil
	} else if err != nil {
		return nil, fmt.Errorf("checking rules file: %w", err)
	}

	status.Installed = true
	status.Message = "graphize rules configured"
	status.Details["size"] = fmt.Sprintf("%d bytes", info.Size())
	status.Details["modified"] = info.ModTime().Format("2006-01-02 15:04:05")

	return status, nil
}

// generateCursorRules creates the content for the Cursor rules file.
func generateCursorRules(graphPath string) string {
	return fmt.Sprintf(`---
description: Graphize knowledge graph integration
globs: ["**/*.go", "**/*.ts", "**/*.java"]
---

# Graphize Integration

This project uses graphize for knowledge graph-based code navigation.

## Available Commands

Use these commands in the terminal:

- `+"`"+`graphize query <node-id>`+"`"+` - Query the knowledge graph
- `+"`"+`graphize explain <node-id>`+"`"+` - Get context about a node
- `+"`"+`graphize report`+"`"+` - Generate analysis report
- `+"`"+`graphize report --health`+"`"+` - Check corpus health

## Graph Path

The knowledge graph is stored at: %s

## Querying the Graph

When exploring code relationships:

1. Use `+"`"+`graphize query func_main --depth 2`+"`"+` to find connected nodes
2. Use `+"`"+`graphize explain type_Server`+"`"+` to understand a node's role
3. Use `+"`"+`graphize report --top 5`+"`"+` to find key architectural nodes

## Node ID Convention

- Functions: `+"`"+`func_filename.go.FunctionName`+"`"+`
- Methods: `+"`"+`method_ReceiverType.MethodName`+"`"+`
- Types: `+"`"+`type_TypeName`+"`"+`
- Packages: `+"`"+`pkg_packagename`+"`"+`
`, graphPath)
}
