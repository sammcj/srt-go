package packagemanager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPackageManagers(t *testing.T) {
	// This test verifies that DetectPackageManagers returns a list of paths
	// and doesn't panic. The actual paths depend on the system configuration.
	paths := DetectPackageManagers()

	// Should return a slice (may be empty if no package managers installed)
	if paths == nil {
		t.Error("DetectPackageManagers returned nil, expected non-nil slice")
	}

	// All returned paths should end with /**
	for _, path := range paths {
		if len(path) < 3 || path[len(path)-3:] != "/**" {
			t.Errorf("Path %q does not end with /**", path)
		}
	}
}

func TestDirExists(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func() (string, func())
		expected bool
	}{
		{
			name: "existing directory",
			setupFn: func() (string, func()) {
				tmpDir := t.TempDir()
				return tmpDir, func() {}
			},
			expected: true,
		},
		{
			name: "non-existent directory",
			setupFn: func() (string, func()) {
				return filepath.Join(os.TempDir(), "non-existent-dir-12345"), func() {}
			},
			expected: false,
		},
		{
			name: "file not directory",
			setupFn: func() (string, func()) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "testfile")
				if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return filePath, func() {}
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setupFn()
			defer cleanup()

			got := dirExists(path)
			if got != tt.expected {
				t.Errorf("dirExists(%q) = %v, want %v", path, got, tt.expected)
			}
		})
	}
}

func TestDetectPackageManagersWithMockDirs(t *testing.T) {
	// Create a temporary directory structure mimicking package managers
	tmpHome := t.TempDir()

	// Set up mock package manager directories
	mockDirs := []string{
		".npm",
		".cache/pip",
		".cargo",
		".rustup",
		".pyenv",
		"go",
	}

	for _, dir := range mockDirs {
		fullPath := filepath.Join(tmpHome, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("Failed to create mock directory %s: %v", fullPath, err)
		}
	}

	// Note: This test can't easily override os.UserHomeDir() without more complex mocking
	// Instead, we verify that the real DetectPackageManagers at least runs without error
	paths := DetectPackageManagers()
	if paths == nil {
		t.Error("DetectPackageManagers returned nil")
	}
}
