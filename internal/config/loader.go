package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Load loads configuration from a file path, creating default if needed
func Load(path string) (*Config, error) {
	// Start with embedded defaults
	cfg, err := DefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	// Determine config file path
	if path == "" {
		// Use default location
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Debug("Could not get home directory, using embedded defaults")
			return cfg, nil
		}

		path = filepath.Join(home, ".srt-settings.json")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config file for user
		if err := createDefaultConfigFile(path); err != nil {
			slog.Debug("Could not create default config file", "path", path, "error", err)
			// Continue with embedded defaults
			return cfg, nil
		}

		slog.Info("Created default configuration file", "path", path)
		// Return the defaults we just created
		return cfg, nil
	}

	// Read existing file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge with defaults (file takes precedence)
	cfg.Merge(&fileCfg)

	// Validate
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func createDefaultConfigFile(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write embedded default config to file with nice formatting
	cfg, err := DefaultConfig()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
