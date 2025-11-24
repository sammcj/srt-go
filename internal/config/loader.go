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

		path = filepath.Join(home, ".srt", "srt-settings.json")
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

// ParseOverrideConfig parses an override configuration from a JSON string or file path
// If input is a valid file path, reads and parses the file
// Otherwise, treats input as inline JSON
func ParseOverrideConfig(input string) (*Config, error) {
	var data []byte

	// Check if input is a file path
	if _, err := os.Stat(input); err == nil {
		// Read from file
		data, err = os.ReadFile(input)
		if err != nil {
			return nil, fmt.Errorf("failed to read override config file: %w", err)
		}
	} else {
		// Treat as inline JSON
		data = []byte(input)
	}

	// Parse JSON
	var override Config
	if err := json.Unmarshal(data, &override); err != nil {
		return nil, fmt.Errorf("invalid override config JSON: %w", err)
	}

	return &override, nil
}

// LoadPreset loads a preset configuration by name
// Presets are stored in the presets/ directory relative to the executable
func LoadPreset(presetName string) (*Config, error) {
	// Get executable path to find presets directory
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	execDir := filepath.Dir(execPath)
	presetPath := filepath.Join(execDir, "presets", presetName+".json")

	// Also check in current working directory for development
	if _, err := os.Stat(presetPath); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		presetPath = filepath.Join(cwd, "presets", presetName+".json")
	}

	// Read preset file
	data, err := os.ReadFile(presetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read preset file %q: %w", presetName, err)
	}

	// Parse JSON
	var presetCfg Config
	if err := json.Unmarshal(data, &presetCfg); err != nil {
		return nil, fmt.Errorf("failed to parse preset file: %w", err)
	}

	return &presetCfg, nil
}

// CreateDefaultConfigFile creates a default configuration file at the specified path
// If path is empty, uses the default location (~/.srt/srt-settings.json)
func CreateDefaultConfigFile(path string) error {
	// Determine config file path
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, ".srt", "srt-settings.json")
	}

	return createDefaultConfigFile(path)
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
