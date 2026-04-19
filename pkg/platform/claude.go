package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ClaudeInstaller integrates graphize with Claude Desktop.
type ClaudeInstaller struct{}

func init() {
	Register(&ClaudeInstaller{})
}

// Name returns "claude".
func (c *ClaudeInstaller) Name() string {
	return "claude"
}

// Description returns the platform description.
func (c *ClaudeInstaller) Description() string {
	return "Claude Desktop MCP server integration"
}

// Install adds graphize MCP server to Claude Desktop configuration.
func (c *ClaudeInstaller) Install(opts InstallOptions) error {
	configPath := claudeConfigPath()
	if configPath == "" {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Load existing config or create new
	config := make(map[string]any)
	if data, err := os.ReadFile(configPath); err == nil { //nolint:gosec // G304: config path is from known location
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing existing config: %w", err)
		}
	}

	// Get or create mcpServers section
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
		config["mcpServers"] = mcpServers
	}

	// Check if already installed
	if _, exists := mcpServers["graphize"]; exists && !opts.Force {
		return fmt.Errorf("graphize already configured in Claude Desktop (use --force to overwrite)")
	}

	// Add graphize MCP server
	graphPath := opts.GraphPath
	if graphPath == "" {
		graphPath = ".graphize"
	}

	mcpServers["graphize"] = map[string]any{
		"command": "graphize",
		"args":    []string{"serve", "--graph", graphPath},
	}

	if opts.DryRun {
		data, _ := json.MarshalIndent(config, "", "  ")
		fmt.Printf("Would write to %s:\n%s\n", configPath, string(data))
		return nil
	}

	// Write config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// Uninstall removes graphize from Claude Desktop configuration.
func (c *ClaudeInstaller) Uninstall(opts InstallOptions) error {
	configPath := claudeConfigPath()
	if configPath == "" {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Load existing config
	data, err := os.ReadFile(configPath) //nolint:gosec // G304: config path is from known location
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Claude Desktop config not found")
		}
		return fmt.Errorf("reading config: %w", err)
	}

	config := make(map[string]any)
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Get mcpServers section
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		return fmt.Errorf("graphize not found in Claude Desktop config")
	}

	// Remove graphize
	if _, exists := mcpServers["graphize"]; !exists {
		return fmt.Errorf("graphize not found in Claude Desktop config")
	}

	delete(mcpServers, "graphize")

	if opts.DryRun {
		data, _ := json.MarshalIndent(config, "", "  ")
		fmt.Printf("Would write to %s:\n%s\n", configPath, string(data))
		return nil
	}

	// Write config
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// Status checks if graphize is configured in Claude Desktop.
func (c *ClaudeInstaller) Status(opts InstallOptions) (*Status, error) {
	configPath := claudeConfigPath()
	if configPath == "" {
		return &Status{
			Installed: false,
			Message:   fmt.Sprintf("unsupported platform: %s", runtime.GOOS),
		}, nil
	}

	status := &Status{
		ConfigPath: configPath,
		Details:    make(map[string]string),
	}

	// Check if config exists
	data, err := os.ReadFile(configPath) //nolint:gosec // G304: config path is from known location
	if err != nil {
		if os.IsNotExist(err) {
			status.Message = "Claude Desktop config not found"
			return status, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	config := make(map[string]any)
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Check for graphize in mcpServers
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		status.Message = "no MCP servers configured"
		return status, nil
	}

	graphizeConfig, exists := mcpServers["graphize"]
	if !exists {
		status.Message = "graphize not configured"
		return status, nil
	}

	status.Installed = true
	status.Message = "graphize MCP server configured"

	// Extract details
	if gc, ok := graphizeConfig.(map[string]any); ok {
		if cmd, ok := gc["command"].(string); ok {
			status.Details["command"] = cmd
		}
		if args, ok := gc["args"].([]any); ok {
			argStrs := make([]string, len(args))
			for i, a := range args {
				argStrs[i] = fmt.Sprintf("%v", a)
			}
			status.Details["args"] = fmt.Sprintf("%v", argStrs)
		}
	}

	return status, nil
}

// claudeConfigPath returns the platform-specific Claude Desktop config path.
func claudeConfigPath() string {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return ""
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json")
	case "linux":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	default:
		return ""
	}
}
