package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Check network defaults
	if cfg.Network.HTTPProxyPort != 0 {
		t.Errorf("Expected HTTPProxyPort to be 0, got %d", cfg.Network.HTTPProxyPort)
	}

	if cfg.Network.SOCKSProxyPort != 0 {
		t.Errorf("Expected SOCKSProxyPort to be 0, got %d", cfg.Network.SOCKSProxyPort)
	}

	// Check filesystem defaults - should have sensible defaults (current dir + package manager caches)
	if len(cfg.Filesystem.AllowWrite) == 0 {
		t.Error("Expected some allowed write paths (current dir and package manager caches)")
	}

	// Check network defaults - should be deny by default
	if cfg.Network.DefaultPolicy != "deny" {
		t.Errorf("Expected defaultPolicy to be 'deny', got %q", cfg.Network.DefaultPolicy)
	}

	if len(cfg.Network.AllowedDomains) != 0 {
		t.Errorf("Expected no allowed domains (most restrictive), got %d", len(cfg.Network.AllowedDomains))
	}

	// Check dangerous patterns exist
	if len(cfg.ScanAndBlockFiles) == 0 {
		t.Error("Expected dangerous file patterns")
	}

	if len(cfg.ScanAndBlockDirs) == 0 {
		t.Error("Expected dangerous directory patterns")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Network: NetworkConfig{
					AllowedDomains: []string{"example.com", "*.github.com"},
					HTTPProxyPort:  8080,
					SOCKSProxyPort: 1080,
				},
				Filesystem: FilesystemConfig{
					DenyRead:   []string{"~/.ssh"},
					AllowWrite: []string{"."},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid domain - overly broad",
			config: &Config{
				Network: NetworkConfig{
					AllowedDomains: []string{"*"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &Config{
				Network: NetworkConfig{
					HTTPProxyPort: 99999,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
