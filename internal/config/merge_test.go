package config

import (
	"reflect"
	"testing"
)

func TestDeepCopy(t *testing.T) {
	original := &Config{
		Network: NetworkConfig{
			DefaultPolicy:  "deny",
			AllowedDomains: []string{"github.com", "npmjs.org"},
			HTTPProxyPort:  8080,
		},
		Filesystem: FilesystemConfig{
			DenyRead:   []string{"~/.ssh/**"},
			AllowWrite: []string{"."},
		},
		Process: ProcessConfig{
			AllowFork:       true,
			AllowSysctlRead: true,
		},
		Verbose: true,
	}

	copy, err := DeepCopy(original)
	if err != nil {
		t.Fatalf("DeepCopy failed: %v", err)
	}

	// Verify copy is equal
	if !reflect.DeepEqual(original.Network, copy.Network) {
		t.Error("Network config not copied correctly")
	}
	if !reflect.DeepEqual(original.Filesystem, copy.Filesystem) {
		t.Error("Filesystem config not copied correctly")
	}

	// Verify runtime fields preserved
	if copy.Verbose != original.Verbose {
		t.Error("Verbose flag not preserved")
	}

	// Verify deep copy (modifying copy shouldn't affect original)
	copy.Network.AllowedDomains[0] = "modified.com"
	if original.Network.AllowedDomains[0] == "modified.com" {
		t.Error("DeepCopy not truly deep - original was modified")
	}
}

func TestMergeConfigs(t *testing.T) {
	tests := []struct {
		name     string
		base     *Config
		override *Config
		expected *Config
	}{
		{
			name: "override empty arrays replace base",
			base: &Config{
				Filesystem: FilesystemConfig{
					AllowWrite: []string{".", "~/.npm/**"},
				},
			},
			override: &Config{
				Filesystem: FilesystemConfig{
					AllowWrite: []string{},
				},
			},
			expected: &Config{
				Filesystem: FilesystemConfig{
					AllowWrite: []string{},
				},
			},
		},
		{
			name: "override non-empty values replace base",
			base: &Config{
				Network: NetworkConfig{
					DefaultPolicy:  "deny",
					AllowedDomains: []string{"github.com"},
				},
			},
			override: &Config{
				Network: NetworkConfig{
					AllowedDomains: []string{"npmjs.org"},
				},
			},
			expected: &Config{
				Network: NetworkConfig{
					// DefaultPolicy will be overridden to empty string since it's in the JSON
					DefaultPolicy:  "",
					AllowedDomains: []string{"npmjs.org"},
				},
			},
		},
		{
			name: "process config override all fields",
			base: &Config{
				Process: ProcessConfig{
					AllowFork:       true,
					AllowSysctlRead: true,
					AllowMachLookup: true,
				},
			},
			override: &Config{
				Process: ProcessConfig{
					AllowFork:       false,
					AllowSysctlRead: false,
					AllowMachLookup: false,
					AllowPosixShm:   false,
				},
			},
			expected: &Config{
				Process: ProcessConfig{
					AllowFork:       false,
					AllowSysctlRead: false,
					AllowMachLookup: false,
					AllowPosixShm:   false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := MergeConfigs(tt.base, tt.override)
			if err != nil {
				t.Fatalf("MergeConfigs failed: %v", err)
			}

			// Compare network config
			if !reflect.DeepEqual(merged.Network, tt.expected.Network) {
				t.Errorf("Network config mismatch:\ngot:  %+v\nwant: %+v", merged.Network, tt.expected.Network)
			}

			// Compare filesystem config
			if !reflect.DeepEqual(merged.Filesystem, tt.expected.Filesystem) {
				t.Errorf("Filesystem config mismatch:\ngot:  %+v\nwant: %+v", merged.Filesystem, tt.expected.Filesystem)
			}

			// Compare process config
			if !reflect.DeepEqual(merged.Process, tt.expected.Process) {
				t.Errorf("Process config mismatch:\ngot:  %+v\nwant: %+v", merged.Process, tt.expected.Process)
			}
		})
	}
}

func TestMergeConfigsEmptyOverride(t *testing.T) {
	base := &Config{
		Network: NetworkConfig{
			DefaultPolicy:  "deny",
			AllowedDomains: []string{"github.com"},
		},
		Filesystem: FilesystemConfig{
			AllowWrite: []string{"."},
		},
	}

	// Empty override will override all fields to their zero values
	override := &Config{}

	merged, err := MergeConfigs(base, override)
	if err != nil {
		t.Fatalf("MergeConfigs failed: %v", err)
	}

	// Verify key fields were overridden
	if merged.Network.DefaultPolicy != "" {
		t.Errorf("Expected empty DefaultPolicy, got %q", merged.Network.DefaultPolicy)
	}
	if len(merged.Network.AllowedDomains) != 0 {
		t.Errorf("Expected empty AllowedDomains, got %v", merged.Network.AllowedDomains)
	}
	if len(merged.Filesystem.AllowWrite) != 0 {
		t.Errorf("Expected empty AllowWrite, got %v", merged.Filesystem.AllowWrite)
	}
}
