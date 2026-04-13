package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage git hooks for automatic graph updates",
	Long: `Install or remove git hooks that automatically update the graph.

Available hooks:
  post-commit   - Run 'graphize analyze' after each commit
  post-checkout - Check if graph is stale after checkout

Examples:
  graphize hook install    # Install hooks
  graphize hook uninstall  # Remove hooks
  graphize hook status     # Check installation status`,
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install git hooks",
	Long:  `Install post-commit and post-checkout hooks for automatic graph updates.`,
	RunE:  runHookInstall,
}

var hookUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove git hooks",
	Long:  `Remove graphize git hooks.`,
	RunE:  runHookUninstall,
}

var hookStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check hook installation status",
	Long:  `Show whether graphize hooks are installed.`,
	RunE:  runHookStatus,
}

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookInstallCmd)
	hookCmd.AddCommand(hookUninstallCmd)
	hookCmd.AddCommand(hookStatusCmd)
}

const postCommitHook = `#!/bin/bash
# Graphize post-commit hook
# Auto-updates the knowledge graph after each commit

# Only run if .graphize exists
if [ -d ".graphize" ]; then
    echo "graphize: Updating knowledge graph..."
    graphize analyze --quiet 2>/dev/null || true
fi
`

const postCheckoutHook = `#!/bin/bash
# Graphize post-checkout hook
# Checks if the graph might be stale after checkout

# Only run if .graphize exists
if [ -d ".graphize" ]; then
    # Check if graph exists
    if [ -d ".graphize/nodes" ]; then
        echo "graphize: Graph may be stale. Run 'graphize analyze' to update."
    fi
fi
`

func runHookInstall(cmd *cobra.Command, args []string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	// Install post-commit hook
	postCommitPath := filepath.Join(hooksDir, "post-commit")
	if err := installHook(postCommitPath, postCommitHook); err != nil {
		return fmt.Errorf("installing post-commit hook: %w", err)
	}
	fmt.Println("Installed post-commit hook")

	// Install post-checkout hook
	postCheckoutPath := filepath.Join(hooksDir, "post-checkout")
	if err := installHook(postCheckoutPath, postCheckoutHook); err != nil {
		return fmt.Errorf("installing post-checkout hook: %w", err)
	}
	fmt.Println("Installed post-checkout hook")

	fmt.Println("\nGraphize hooks installed successfully.")
	fmt.Println("The graph will auto-update on commit and warn about staleness on checkout.")

	return nil
}

func runHookUninstall(cmd *cobra.Command, args []string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitDir, "hooks")

	// Remove post-commit hook if it's ours
	postCommitPath := filepath.Join(hooksDir, "post-commit")
	if removed, err := removeHookIfOurs(postCommitPath); err != nil {
		return fmt.Errorf("removing post-commit hook: %w", err)
	} else if removed {
		fmt.Println("Removed post-commit hook")
	}

	// Remove post-checkout hook if it's ours
	postCheckoutPath := filepath.Join(hooksDir, "post-checkout")
	if removed, err := removeHookIfOurs(postCheckoutPath); err != nil {
		return fmt.Errorf("removing post-checkout hook: %w", err)
	} else if removed {
		fmt.Println("Removed post-checkout hook")
	}

	fmt.Println("\nGraphize hooks removed.")

	return nil
}

func runHookStatus(cmd *cobra.Command, args []string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitDir, "hooks")

	fmt.Println("Graphize Git Hook Status")
	fmt.Println("========================")
	fmt.Println()

	// Check post-commit
	postCommitPath := filepath.Join(hooksDir, "post-commit")
	postCommitStatus := checkHookStatus(postCommitPath)
	fmt.Printf("post-commit:   %s\n", postCommitStatus)

	// Check post-checkout
	postCheckoutPath := filepath.Join(hooksDir, "post-checkout")
	postCheckoutStatus := checkHookStatus(postCheckoutPath)
	fmt.Printf("post-checkout: %s\n", postCheckoutStatus)

	return nil
}

func findGitDir() (string, error) {
	// Use git rev-parse to find .git directory
	out, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git not installed)")
	}

	gitDir := filepath.Clean(string(out[:len(out)-1])) // Remove trailing newline
	if !filepath.IsAbs(gitDir) {
		cwd, _ := os.Getwd()
		gitDir = filepath.Join(cwd, gitDir)
	}

	return gitDir, nil
}

func installHook(path, content string) error {
	// Check if hook already exists
	if _, err := os.Stat(path); err == nil {
		// Read existing hook
		existing, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Check if it's already our hook
		if string(existing) == content {
			return nil // Already installed
		}

		// Check if it contains our marker
		if containsGraphizeMarker(string(existing)) {
			// Replace our hook
			return os.WriteFile(path, []byte(content), 0755) //nolint:gosec // G306: git hooks must be executable
		}

		// Append to existing hook
		combined := string(existing) + "\n" + content
		return os.WriteFile(path, []byte(combined), 0755) //nolint:gosec // G306: git hooks must be executable
	}

	// Create new hook
	return os.WriteFile(path, []byte(content), 0755) //nolint:gosec // G306: git hooks must be executable
}

func removeHookIfOurs(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !containsGraphizeMarker(string(content)) {
		return false, nil
	}

	// If the entire file is our hook, remove it
	if string(content) == postCommitHook || string(content) == postCheckoutHook {
		return true, os.Remove(path)
	}

	// TODO: Handle case where our hook is appended to another hook
	// For now, just warn the user
	fmt.Printf("Warning: %s contains graphize hook mixed with other hooks.\n", path)
	fmt.Println("Please manually remove the graphize section.")

	return false, nil
}

func containsGraphizeMarker(content string) bool {
	return len(content) > 0 && (
	// Check for our comment markers
	(len(content) >= 26 && content[0:26] == "# Graphize post-commit") ||
		(len(content) >= 28 && content[0:28] == "# Graphize post-checkout") ||
		// Or contains graphize command
		(len(content) > 8 && stringContains(content, "graphize analyze")) ||
		(len(content) > 8 && stringContains(content, "graphize:")))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func checkHookStatus(path string) string {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "not installed"
	}
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	if containsGraphizeMarker(string(content)) {
		return "installed"
	}

	return "not installed (hook exists but not graphize)"
}
