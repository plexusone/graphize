package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GeminiInstaller integrates graphize with Gemini CLI.
type GeminiInstaller struct{}

func init() {
	Register(&GeminiInstaller{})
}

// Name returns "gemini".
func (g *GeminiInstaller) Name() string {
	return "gemini"
}

// Description returns the platform description.
func (g *GeminiInstaller) Description() string {
	return "Gemini CLI context file integration"
}

// Install adds graphize context to Gemini configuration.
func (g *GeminiInstaller) Install(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	geminiDir := filepath.Join(projectPath, ".gemini")
	contextFile := filepath.Join(geminiDir, "context.yaml")

	// Ensure directory exists
	if err := os.MkdirAll(geminiDir, 0755); err != nil {
		return fmt.Errorf("creating .gemini directory: %w", err)
	}

	// Load existing config or create new
	config := make(map[string]any)
	if data, err := os.ReadFile(contextFile); err == nil { //nolint:gosec // G304: path is from known location
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing existing config: %w", err)
		}
	}

	// Check if already installed
	if tools, ok := config["tools"].([]any); ok {
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

	// Add or update graphize tool
	config = updateGeminiConfig(config, graphPath, opts.Force)

	if opts.DryRun {
		data, _ := yaml.Marshal(config)
		fmt.Printf("Would write to %s:\n%s\n", contextFile, string(data))
		return nil
	}

	// Write config
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(contextFile, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// Uninstall removes graphize from Gemini configuration.
func (g *GeminiInstaller) Uninstall(opts InstallOptions) error {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	contextFile := filepath.Join(projectPath, ".gemini", "context.yaml")

	data, err := os.ReadFile(contextFile) //nolint:gosec // G304: path is from known location
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Gemini config not found")
		}
		return fmt.Errorf("reading config: %w", err)
	}

	config := make(map[string]any)
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Remove graphize from tools
	tools, ok := config["tools"].([]any)
	if !ok {
		return fmt.Errorf("graphize not found in Gemini config")
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
		return fmt.Errorf("graphize not found in Gemini config")
	}

	config["tools"] = newTools

	if opts.DryRun {
		data, _ := yaml.Marshal(config)
		fmt.Printf("Would write to %s:\n%s\n", contextFile, string(data))
		return nil
	}

	data, err = yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(contextFile, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// Status checks if graphize is configured for Gemini.
func (g *GeminiInstaller) Status(opts InstallOptions) (*Status, error) {
	projectPath := opts.ProjectPath
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	contextFile := filepath.Join(projectPath, ".gemini", "context.yaml")

	status := &Status{
		ConfigPath: contextFile,
		Details:    make(map[string]string),
	}

	data, err := os.ReadFile(contextFile) //nolint:gosec // G304: path is from known location
	if os.IsNotExist(err) {
		status.Message = "Gemini config not found"
		return status, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	config := make(map[string]any)
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Check for graphize in tools
	tools, ok := config["tools"].([]any)
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

// updateGeminiConfig adds or updates graphize in the config.
func updateGeminiConfig(config map[string]any, graphPath string, force bool) map[string]any {
	graphizeTool := map[string]any{
		"name":        "graphize",
		"description": "Knowledge graph-based code navigation",
		"commands": []map[string]any{
			{
				"name":        "query",
				"description": "Query the knowledge graph",
				"usage":       "graphize query <node-id> [--depth N]",
			},
			{
				"name":        "explain",
				"description": "Get detailed context about a node",
				"usage":       "graphize explain <node-id>",
			},
			{
				"name":        "report",
				"description": "Generate architecture analysis report",
				"usage":       "graphize report [--health]",
			},
		},
		"graph_path": graphPath,
	}

	tools, ok := config["tools"].([]any)
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
	config["tools"] = tools

	// Add context note if not present
	if _, ok := config["notes"]; !ok {
		config["notes"] = []string{
			"Use graphize commands to navigate the codebase knowledge graph",
			fmt.Sprintf("Graph location: %s", graphPath),
		}
	} else if notes, ok := config["notes"].([]any); ok {
		hasGraphize := false
		for _, n := range notes {
			if s, ok := n.(string); ok && strings.Contains(s, "graphize") {
				hasGraphize = true
				break
			}
		}
		if !hasGraphize {
			notes = append(notes, "Use graphize commands to navigate the codebase knowledge graph")
			config["notes"] = notes
		}
	}

	return config
}
