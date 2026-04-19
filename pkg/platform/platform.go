// Package platform provides installers for integrating graphize with AI coding platforms.
package platform

import (
	"fmt"
	"sort"
	"sync"
)

// Installer defines the interface for platform integration installers.
type Installer interface {
	// Name returns the platform name (e.g., "claude", "cursor").
	Name() string

	// Description returns a brief description of the platform.
	Description() string

	// Install integrates graphize with the platform.
	Install(opts InstallOptions) error

	// Uninstall removes graphize integration from the platform.
	Uninstall(opts InstallOptions) error

	// Status checks the current installation status.
	Status(opts InstallOptions) (*Status, error)
}

// InstallOptions configures the installation process.
type InstallOptions struct {
	// GraphPath is the path to the .graphize directory.
	GraphPath string

	// ProjectPath is the path to the project root.
	ProjectPath string

	// Force overwrites existing configurations.
	Force bool

	// DryRun shows what would be done without making changes.
	DryRun bool
}

// Status represents the installation status for a platform.
type Status struct {
	// Installed indicates if the integration is currently installed.
	Installed bool `json:"installed"`

	// ConfigPath is the path to the configuration file.
	ConfigPath string `json:"config_path,omitempty"`

	// Version is the graphize version in the configuration.
	Version string `json:"version,omitempty"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`

	// Details contains platform-specific status information.
	Details map[string]string `json:"details,omitempty"`
}

// Registry manages platform installers.
var (
	installers   = make(map[string]Installer)
	installersMu sync.RWMutex
)

// Register adds an installer to the registry.
func Register(installer Installer) {
	installersMu.Lock()
	defer installersMu.Unlock()
	installers[installer.Name()] = installer
}

// Get returns an installer by name.
// Returns nil if the installer is not registered.
func Get(name string) Installer {
	installersMu.RLock()
	defer installersMu.RUnlock()
	return installers[name]
}

// List returns all registered installer names.
func List() []string {
	installersMu.RLock()
	defer installersMu.RUnlock()

	names := make([]string, 0, len(installers))
	for name := range installers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListWithDescriptions returns all installers with their descriptions.
func ListWithDescriptions() []InstallerInfo {
	installersMu.RLock()
	defer installersMu.RUnlock()

	infos := make([]InstallerInfo, 0, len(installers))
	for _, inst := range installers {
		infos = append(infos, InstallerInfo{
			Name:        inst.Name(),
			Description: inst.Description(),
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// InstallerInfo contains basic installer information.
type InstallerInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Install runs the installer for a platform.
func Install(name string, opts InstallOptions) error {
	inst := Get(name)
	if inst == nil {
		return fmt.Errorf("unknown platform: %s", name)
	}
	return inst.Install(opts)
}

// Uninstall removes the integration for a platform.
func Uninstall(name string, opts InstallOptions) error {
	inst := Get(name)
	if inst == nil {
		return fmt.Errorf("unknown platform: %s", name)
	}
	return inst.Uninstall(opts)
}

// CheckStatus checks the installation status for a platform.
func CheckStatus(name string, opts InstallOptions) (*Status, error) {
	inst := Get(name)
	if inst == nil {
		return nil, fmt.Errorf("unknown platform: %s", name)
	}
	return inst.Status(opts)
}
