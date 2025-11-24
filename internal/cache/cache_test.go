package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheSaveLoad(t *testing.T) {
	// Create temporary cache
	tmpDir := t.TempDir()
	originalEnv := os.Getenv("USER")
	os.Setenv("USER", "testuser")
	defer os.Setenv("USER", originalEnv)

	// Override cache path to use temp directory
	cachePath := filepath.Join(tmpDir, ".srt-cache-testuser.json")

	// Save cache
	if err := os.WriteFile(cachePath, []byte(`{
		"packageManagerPaths": ["/opt/homebrew/**", "~/.npm/**"],
		"configMtime": "2025-01-01T00:00:00Z",
		"timestamp": "2025-01-01T00:00:00Z"
	}`), 0600); err != nil {
		t.Fatalf("Failed to write test cache: %v", err)
	}

	// Test that we can determine cache path
	_, err := GetCachePath()
	if err != nil {
		t.Errorf("GetCachePath() failed: %v", err)
	}
}

func TestCacheValidity(t *testing.T) {
	tests := []struct {
		name       string
		cache      *PathCache
		configPath string
		expected   bool
	}{
		{
			name:     "nil cache is invalid",
			cache:    nil,
			expected: false,
		},
		{
			name: "fresh cache is valid",
			cache: &PathCache{
				PackageManagerPaths: []string{"/opt/homebrew/**"},
				Timestamp:           time.Now(),
			},
			expected: true,
		},
		{
			name: "expired cache is invalid",
			cache: &PathCache{
				PackageManagerPaths: []string{"/opt/homebrew/**"},
				Timestamp:           time.Now().Add(-2 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.IsValid(tt.configPath)
			if result != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCacheClear(t *testing.T) {
	tmpDir := t.TempDir()
	originalEnv := os.Getenv("USER")
	os.Setenv("USER", "testuser")
	defer os.Setenv("USER", originalEnv)

	cachePath := filepath.Join(tmpDir, ".srt-cache-testuser.json")

	// Create a cache file
	if err := os.WriteFile(cachePath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test cache file: %v", err)
	}

	// Clear shouldn't fail even if file doesn't exist
	if err := Clear(); err != nil {
		t.Errorf("Clear() with non-existent file failed: %v", err)
	}
}

func TestGetConfigMtime(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	// Non-existent file should return zero time
	mtime := GetConfigMtime(configPath)
	if !mtime.IsZero() {
		t.Errorf("Expected zero time for non-existent file, got %v", mtime)
	}

	// Create file
	if err := os.WriteFile(configPath, []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should return non-zero time
	mtime = GetConfigMtime(configPath)
	if mtime.IsZero() {
		t.Error("Expected non-zero time for existing file")
	}

	// Empty path should return zero time
	mtime = GetConfigMtime("")
	if !mtime.IsZero() {
		t.Errorf("Expected zero time for empty path, got %v", mtime)
	}
}
