package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CodexInstaller integrates graphize with OpenAI Codex CLI.
type CodexInstaller struct{}

func init() {
	Register(&CodexInstaller{})
}

// Name returns "codex".
func (c *CodexInstaller) Name() string {
	return "codex"
}

// Description returns the platform description.
func (c *CodexInstaller) Description() string {
	return "OpenAI Codex CLI hooks integration"
}

// Install adds graphize hooks to Codex configuration.
func (c *CodexInstaller) Install(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	hooksFile := filepath.Join(projectPath, "hooks.json")

	// Load existing hooks or create new
	hooks := make(map[string]any)
	if data, err := os.ReadFile(hooksFile); err == nil { //nolint:gosec // G304: path is from known location
		if err := json.Unmarshal(data, &hooks); err != nil {
			return fmt.Errorf("parsing existing hooks: %w", err)
		}
	}

	// Check if already installed
	if tools, ok := hooks["tools"].([]any); ok {
		for _, t := range tools {
			if tm, ok := t.(map[string]any); ok {
				if tm["name"] == "graphize" && !opts.Force {
					return fmt.Errorf("graphize already configured (use --force to overwrite)")
				}
			}
		}
	}

	graphPath := opts.GraphPath
	if graphPath == "" {
		graphPath = ".graphize"
	}

	// Add graphize tool
	hooks = updateCodexHooks(hooks, graphPath, opts.Force)

	if opts.DryRun {
		data, _ := json.MarshalIndent(hooks, "", "  ")
		fmt.Printf("Would write to %s:\n%s\n", hooksFile, string(data))
		return nil
	}

	// Write hooks
	data, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling hooks: %w", err)
	}

	if err := os.WriteFile(hooksFile, data, 0600); err != nil {
		return fmt.Errorf("writing hooks: %w", err)
	}

	return nil
}

// Uninstall removes graphize from Codex configuration.
func (c *CodexInstaller) Uninstall(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	hooksFile := filepath.Join(projectPath, "hooks.json")

	data, err := os.ReadFile(hooksFile) //nolint:gosec // G304: path is from known location
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("hooks.json not found")
		}
		return fmt.Errorf("reading hooks: %w", err)
	}

	hooks := make(map[string]any)
	if err := json.Unmarshal(data, &hooks); err != nil {
		return fmt.Errorf("parsing hooks: %w", err)
	}

	// Remove graphize from tools
	tools, ok := hooks["tools"].([]any)
	if !ok {
		return fmt.Errorf("graphize not found in hooks.json")
	}

	var newTools []any
	found := false
	for _, t := range tools {
		if tm, ok := t.(map[string]any); ok {
			if tm["name"] == "graphize" {
				found = true
				continue
			}
		}
		newTools = append(newTools, t)
	}

	if !found {
		return fmt.Errorf("graphize not found in hooks.json")
	}

	hooks["tools"] = newTools

	if opts.DryRun {
		data, _ := json.MarshalIndent(hooks, "", "  ")
		fmt.Printf("Would write to %s:\n%s\n", hooksFile, string(data))
		return nil
	}

	data, err = json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling hooks: %w", err)
	}

	if err := os.WriteFile(hooksFile, data, 0600); err != nil {
		return fmt.Errorf("writing hooks: %w", err)
	}

	return nil
}

// Status checks if graphize is configured for Codex.
func (c *CodexInstaller) Status(opts InstallOptions) (*Status, error) {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	hooksFile := filepath.Join(projectPath, "hooks.json")

	status := &Status{
		ConfigPath: hooksFile,
		Details:    make(map[string]string),
	}

	data, err := os.ReadFile(hooksFile) //nolint:gosec // G304: path is from known location
	if os.IsNotExist(err) {
		status.Message = "hooks.json not found"
		return status, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading hooks: %w", err)
	}

	hooks := make(map[string]any)
	if err := json.Unmarshal(data, &hooks); err != nil {
		return nil, fmt.Errorf("parsing hooks: %w", err)
	}

	// Check for graphize in tools
	tools, ok := hooks["tools"].([]any)
	if !ok {
		status.Message = "no tools configured"
		return status, nil
	}

	for _, t := range tools {
		if tm, ok := t.(map[string]any); ok {
			if tm["name"] == "graphize" {
				status.Installed = true
				status.Message = "graphize tool configured"
				if desc, ok := tm["description"].(string); ok {
					status.Details["description"] = desc
				}
				return status, nil
			}
		}
	}

	status.Message = "graphize not configured"
	return status, nil
}

// updateCodexHooks adds or updates graphize in the hooks config.
func updateCodexHooks(hooks map[string]any, graphPath string, force bool) map[string]any {
	graphizeTool := map[string]any{
		"name":        "graphize",
		"description": "Query the codebase knowledge graph",
		"commands": []map[string]string{
			{
				"name":    "query",
				"command": "graphize query",
				"args":    "<node-id> [--depth N]",
			},
			{
				"name":    "explain",
				"command": "graphize explain",
				"args":    "<node-id>",
			},
			{
				"name":    "report",
				"command": "graphize report",
				"args":    "[--health] [--top N]",
			},
		},
		"graph_path": graphPath,
	}

	tools, ok := hooks["tools"].([]any)
	if !ok {
		tools = []any{}
	}

	if force {
		// Remove existing graphize
		var newTools []any
		for _, t := range tools {
			if tm, ok := t.(map[string]any); ok {
				if tm["name"] == "graphize" {
					continue
				}
			}
			newTools = append(newTools, t)
		}
		tools = newTools
	}

	tools = append(tools, graphizeTool)
	hooks["tools"] = tools

	return hooks
}
