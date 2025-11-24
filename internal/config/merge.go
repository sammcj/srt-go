package config

import "encoding/json"

// DeepCopy creates a deep copy of a Config
func DeepCopy(cfg *Config) (*Config, error) {
	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	var copy Config
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}

	// Preserve runtime fields that aren't in JSON
	copy.Verbose = cfg.Verbose

	return &copy, nil
}

// MergeConfigs merges base and override configurations with proper nil/empty handling
// Merge semantics:
// - nil values in override = keep base value
// - empty arrays in override = override with empty (most restrictive)
// - non-empty values in override = replace base value
func MergeConfigs(base, override *Config) (*Config, error) {
	// Deep copy base to avoid modifying it
	merged, err := DeepCopy(base)
	if err != nil {
		return nil, err
	}

	// To distinguish nil from empty arrays, we need to parse override as raw JSON
	// and check which fields were explicitly set
	overrideJSON, err := json.Marshal(override)
	if err != nil {
		return nil, err
	}

	var overrideMap map[string]interface{}
	if err := json.Unmarshal(overrideJSON, &overrideMap); err != nil {
		return nil, err
	}

	// Merge network settings
	if networkMap, ok := overrideMap["network"].(map[string]interface{}); ok {
		mergeNetworkConfig(&merged.Network, &override.Network, networkMap)
	}

	// Merge filesystem settings
	if fsMap, ok := overrideMap["filesystem"].(map[string]interface{}); ok {
		mergeFilesystemConfig(&merged.Filesystem, &override.Filesystem, fsMap)
	}

	// Merge process settings
	if processMap, ok := overrideMap["process"].(map[string]interface{}); ok {
		mergeProcessConfig(&merged.Process, &override.Process, processMap)
	}

	// Merge other fields if explicitly set
	if _, ok := overrideMap["scanAndBlockFiles"]; ok {
		merged.ScanAndBlockFiles = override.ScanAndBlockFiles
	}
	if _, ok := overrideMap["scanAndBlockDirs"]; ok {
		merged.ScanAndBlockDirs = override.ScanAndBlockDirs
	}
	if _, ok := overrideMap["ignoreViolations"]; ok {
		merged.Violations = override.Violations
	}
	if ripgrepMap, ok := overrideMap["ripgrep"].(map[string]interface{}); ok {
		if _, hasCommand := ripgrepMap["command"]; hasCommand {
			merged.Ripgrep.Command = override.Ripgrep.Command
		}
		if _, hasArgs := ripgrepMap["args"]; hasArgs {
			merged.Ripgrep.Args = override.Ripgrep.Args
		}
	}

	return merged, nil
}

func mergeNetworkConfig(base, override *NetworkConfig, overrideMap map[string]interface{}) {
	if _, ok := overrideMap["defaultPolicy"]; ok {
		base.DefaultPolicy = override.DefaultPolicy
	}
	if _, ok := overrideMap["allowedDomains"]; ok {
		base.AllowedDomains = override.AllowedDomains
	}
	if _, ok := overrideMap["deniedDomains"]; ok {
		base.DeniedDomains = override.DeniedDomains
	}
	if _, ok := overrideMap["allowUnixSockets"]; ok {
		base.AllowUnixSockets = override.AllowUnixSockets
	}
	if _, ok := overrideMap["allowLocalBinding"]; ok {
		base.AllowLocalBinding = override.AllowLocalBinding
	}
	if _, ok := overrideMap["httpProxyPort"]; ok {
		base.HTTPProxyPort = override.HTTPProxyPort
	}
	if _, ok := overrideMap["socksProxyPort"]; ok {
		base.SOCKSProxyPort = override.SOCKSProxyPort
	}
}

func mergeFilesystemConfig(base, override *FilesystemConfig, overrideMap map[string]interface{}) {
	if _, ok := overrideMap["denyRead"]; ok {
		base.DenyRead = override.DenyRead
	}
	if _, ok := overrideMap["allowWrite"]; ok {
		base.AllowWrite = override.AllowWrite
	}
	if _, ok := overrideMap["denyWrite"]; ok {
		base.DenyWrite = override.DenyWrite
	}
	if _, ok := overrideMap["allowUnlink"]; ok {
		base.AllowUnlink = override.AllowUnlink
	}
}

func mergeProcessConfig(base, override *ProcessConfig, overrideMap map[string]interface{}) {
	if _, ok := overrideMap["allowFork"]; ok {
		base.AllowFork = override.AllowFork
	}
	if _, ok := overrideMap["allowSysctlRead"]; ok {
		base.AllowSysctlRead = override.AllowSysctlRead
	}
	if _, ok := overrideMap["allowMachLookup"]; ok {
		base.AllowMachLookup = override.AllowMachLookup
	}
	if _, ok := overrideMap["allowPosixShm"]; ok {
		base.AllowPosixShm = override.AllowPosixShm
	}
}
