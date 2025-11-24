package config

import (
	_ "embed"
	"encoding/json"
)

//go:embed default-config.json
var defaultConfigJSON []byte

// Config represents the sandbox configuration
type Config struct {
	Network           NetworkConfig       `json:"network"`
	Filesystem        FilesystemConfig    `json:"filesystem"`
	Process           ProcessConfig       `json:"process"`
	ScanAndBlockFiles []string            `json:"scanAndBlockFiles"`
	ScanAndBlockDirs  []string            `json:"scanAndBlockDirs"`
	Violations        map[string][]string `json:"ignoreViolations"`
	Ripgrep           RipgrepConfig       `json:"ripgrep"`
	Verbose           bool                `json:"-"` // Not from JSON
}

// NetworkConfig contains network-related settings
type NetworkConfig struct {
	DefaultPolicy     string   `json:"defaultPolicy"` // "allow" or "deny"
	AllowedDomains    []string `json:"allowedDomains"`
	DeniedDomains     []string `json:"deniedDomains"`
	AllowUnixSockets  []string `json:"allowUnixSockets"`
	AllowLocalBinding bool     `json:"allowLocalBinding"`
	HTTPProxyPort     int      `json:"httpProxyPort"`
	SOCKSProxyPort    int      `json:"socksProxyPort"`
}

// FilesystemConfig contains filesystem-related settings
type FilesystemConfig struct {
	DenyRead    []string `json:"denyRead"`
	AllowWrite  []string `json:"allowWrite"`
	DenyWrite   []string `json:"denyWrite"`
	AllowUnlink []string `json:"allowUnlink"` // Paths where file deletion/moving is allowed
}

// ProcessConfig contains process-related sandbox permissions
type ProcessConfig struct {
	AllowFork       bool `json:"allowFork"`       // Allow process forking
	AllowSysctlRead bool `json:"allowSysctlRead"` // Allow reading system information
	AllowMachLookup bool `json:"allowMachLookup"` // Allow Mach IPC lookups
	AllowPosixShm   bool `json:"allowPosixShm"`   // Allow POSIX shared memory
}

// RipgrepConfig contains ripgrep-specific settings
type RipgrepConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// DefaultConfig returns a configuration from the embedded default
func DefaultConfig() (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(defaultConfigJSON, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Merge merges another config into this one (other takes precedence)
func (c *Config) Merge(other *Config) {
	if other.Network.DefaultPolicy != "" {
		c.Network.DefaultPolicy = other.Network.DefaultPolicy
	}
	if len(other.Network.AllowedDomains) > 0 {
		c.Network.AllowedDomains = other.Network.AllowedDomains
	}
	if len(other.Network.DeniedDomains) > 0 {
		c.Network.DeniedDomains = other.Network.DeniedDomains
	}
	if len(other.Network.AllowUnixSockets) > 0 {
		c.Network.AllowUnixSockets = other.Network.AllowUnixSockets
	}
	if other.Network.HTTPProxyPort != 0 {
		c.Network.HTTPProxyPort = other.Network.HTTPProxyPort
	}
	if other.Network.SOCKSProxyPort != 0 {
		c.Network.SOCKSProxyPort = other.Network.SOCKSProxyPort
	}
	if len(other.Filesystem.DenyRead) > 0 {
		c.Filesystem.DenyRead = other.Filesystem.DenyRead
	}
	if len(other.Filesystem.AllowWrite) > 0 {
		c.Filesystem.AllowWrite = other.Filesystem.AllowWrite
	}
	if len(other.Filesystem.DenyWrite) > 0 {
		c.Filesystem.DenyWrite = other.Filesystem.DenyWrite
	}
	if len(other.Filesystem.AllowUnlink) > 0 {
		c.Filesystem.AllowUnlink = other.Filesystem.AllowUnlink
	}
	if len(other.ScanAndBlockFiles) > 0 {
		c.ScanAndBlockFiles = other.ScanAndBlockFiles
	}
	if len(other.ScanAndBlockDirs) > 0 {
		c.ScanAndBlockDirs = other.ScanAndBlockDirs
	}
	if len(other.Violations) > 0 {
		c.Violations = other.Violations
	}
	if other.Ripgrep.Command != "" {
		c.Ripgrep.Command = other.Ripgrep.Command
	}
	if len(other.Ripgrep.Args) > 0 {
		c.Ripgrep.Args = other.Ripgrep.Args
	}
	// Process permissions - only merge if at least one is true (indicates explicit configuration)
	// This prevents false defaults from overwriting true defaults when process section is missing
	if other.Process.AllowFork || other.Process.AllowSysctlRead || other.Process.AllowMachLookup || other.Process.AllowPosixShm {
		c.Process.AllowFork = other.Process.AllowFork
		c.Process.AllowSysctlRead = other.Process.AllowSysctlRead
		c.Process.AllowMachLookup = other.Process.AllowMachLookup
		c.Process.AllowPosixShm = other.Process.AllowPosixShm
	}
}
