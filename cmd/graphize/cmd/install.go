package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/plexusone/graphize/pkg/platform"
	"github.com/spf13/cobra"
)

var (
	installUninstall bool
	installStatus    bool
	installForce     bool
	installDryRun    bool
	installJSON      bool
)

var installCmd = &cobra.Command{
	Use:   "install <platform>",
	Short: "Install graphize integration for AI coding platforms",
	Long: `Install graphize integration for various AI coding platforms.

Supported platforms:
  - claude    : Claude Desktop MCP server configuration
  - cursor    : Cursor IDE rules file
  - copilot   : GitHub Copilot skills
  - aider     : Aider AGENTS.md section
  - gemini    : Gemini CLI context file
  - codex     : OpenAI Codex CLI hooks

Examples:
  graphize install claude          # Install Claude Desktop MCP server
  graphize install cursor          # Add Cursor rules file
  graphize install --status claude # Check installation status
  graphize install --uninstall cursor  # Remove integration
  graphize install --dry-run aider # Preview what would be done

Use 'graphize install' without arguments to list all platforms.`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&installUninstall, "uninstall", false, "Remove the integration")
	installCmd.Flags().BoolVar(&installStatus, "status", false, "Check installation status")
	installCmd.Flags().BoolVar(&installForce, "force", false, "Overwrite existing configuration")
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Show what would be done without making changes")
	installCmd.Flags().BoolVar(&installJSON, "json", false, "Output status in JSON format")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// List platforms if no arguments
	if len(args) == 0 {
		return listPlatforms()
	}

	platformName := strings.ToLower(args[0])

	// Resolve paths
	absGraphPath, err := filepath.Abs(graphPath)
	if err != nil {
		return fmt.Errorf("resolving graph path: %w", err)
	}

	projectPath, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("resolving project path: %w", err)
	}

	opts := platform.InstallOptions{
		GraphPath:   absGraphPath,
		ProjectPath: projectPath,
		Force:       installForce,
		DryRun:      installDryRun,
	}

	// Status check
	if installStatus {
		return checkPlatformStatus(platformName, opts)
	}

	// Uninstall
	if installUninstall {
		return uninstallPlatform(platformName, opts)
	}

	// Install
	return installPlatform(platformName, opts)
}

func listPlatforms() error {
	infos := platform.ListWithDescriptions()

	if installJSON {
		data, err := json.MarshalIndent(infos, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("Available platforms:")
	fmt.Println()
	for _, info := range infos {
		fmt.Printf("  %-10s  %s\n", info.Name, info.Description)
	}
	fmt.Println()
	fmt.Println("Usage: graphize install <platform>")
	fmt.Println("       graphize install --status <platform>")
	fmt.Println("       graphize install --uninstall <platform>")

	return nil
}

func installPlatform(name string, opts platform.InstallOptions) error {
	inst := platform.Get(name)
	if inst == nil {
		return fmt.Errorf("unknown platform: %s\n\nUse 'graphize install' to list available platforms.", name)
	}

	if opts.DryRun {
		fmt.Printf("[DRY RUN] Installing graphize for %s\n\n", name)
	}

	if err := inst.Install(opts); err != nil {
		return err
	}

	if !opts.DryRun {
		fmt.Printf("Successfully installed graphize for %s\n", name)

		// Show post-install instructions
		switch name {
		case "claude":
			fmt.Println("\nRestart Claude Desktop to load the MCP server.")
		case "cursor":
			fmt.Println("\nThe rules file is now available in .cursor/rules/")
		case "aider":
			fmt.Println("\nThe AGENTS.md file has been updated.")
		}
	}

	return nil
}

func uninstallPlatform(name string, opts platform.InstallOptions) error {
	inst := platform.Get(name)
	if inst == nil {
		return fmt.Errorf("unknown platform: %s", name)
	}

	if opts.DryRun {
		fmt.Printf("[DRY RUN] Uninstalling graphize for %s\n\n", name)
	}

	if err := inst.Uninstall(opts); err != nil {
		return err
	}

	if !opts.DryRun {
		fmt.Printf("Successfully uninstalled graphize for %s\n", name)
	}

	return nil
}

func checkPlatformStatus(name string, opts platform.InstallOptions) error {
	inst := platform.Get(name)
	if inst == nil {
		return fmt.Errorf("unknown platform: %s", name)
	}

	status, err := inst.Status(opts)
	if err != nil {
		return err
	}

	if installJSON {
		data, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	fmt.Printf("Platform: %s\n", name)
	fmt.Printf("Status: ")
	if status.Installed {
		fmt.Println("INSTALLED")
	} else {
		fmt.Println("NOT INSTALLED")
	}

	if status.ConfigPath != "" {
		fmt.Printf("Config: %s\n", status.ConfigPath)
	}

	if status.Message != "" {
		fmt.Printf("Message: %s\n", status.Message)
	}

	if len(status.Details) > 0 {
		fmt.Println("Details:")
		for k, v := range status.Details {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	return nil
}
