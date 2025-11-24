package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasBalancedParentheses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "balanced parentheses",
			input:    "(version 1) (allow file-read*) (deny network*)",
			expected: true,
		},
		{
			name:     "unbalanced - missing closing",
			input:    "(version 1) (allow file-read*",
			expected: false,
		},
		{
			name:     "unbalanced - extra closing",
			input:    "(version 1)) (allow file-read*)",
			expected: false,
		},
		{
			name:     "nested balanced",
			input:    "(allow (subpath \"/home\"))",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasBalancedParentheses(tt.input)
			if got != tt.expected {
				t.Errorf("hasBalancedParentheses(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateProfile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid profile",
			content: `(version 1)
(deny default)
(allow file-read*)
(allow process-exec*)`,
			shouldError: false,
		},
		{
			name:        "missing version declaration",
			content:     "(deny default)\n(allow file-read*)",
			shouldError: true,
			errorMsg:    "version 1",
		},
		{
			name:        "unbalanced parentheses",
			content:     "(version 1)\n(deny default\n(allow file-read*)",
			shouldError: true,
			errorMsg:    "unbalanced parentheses",
		},
		{
			name:        "no deny or allow statements",
			content:     "(version 1)\n",
			shouldError: true,
			errorMsg:    "deny/allow statements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary profile file
			tmpDir := t.TempDir()
			profilePath := filepath.Join(tmpDir, "test-profile.sb")
			if err := os.WriteFile(profilePath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("Failed to create test profile: %v", err)
			}

			err := ValidateProfile(profilePath)
			if tt.shouldError {
				if err == nil {
					t.Errorf("ValidateProfile() expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateProfile() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateProfile() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateSeatbeltProfile(t *testing.T) {
	tests := []struct {
		name             string
		httpPort         int
		socksPort        int
		denyReadPaths    []string
		allowWritePaths  []string
		denyWritePaths   []string
		allowUnlinkPaths []string
		allowFork        bool
		allowSysctlRead  bool
		allowMachLookup  bool
		allowPosixShm    bool
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:            "basic profile with proxy",
			httpPort:        8080,
			socksPort:       1080,
			denyReadPaths:   []string{"/home/user/.ssh"},
			allowWritePaths: []string{"/tmp"},
			allowFork:       true,
			wantContains: []string{
				"(version 1)",
				"(allow process-fork)",
				"(deny file-read* (subpath \"/home/user/.ssh\"))",
				"(allow file-write* (subpath \"/tmp\"))",
				"localhost:8080",
				"localhost:1080",
			},
		},
		{
			name:            "no fork permission with proxy",
			httpPort:        8080,
			socksPort:       1080,
			allowFork:       false,
			allowSysctlRead: false,
			wantContains: []string{
				"(version 1)",
				"localhost:8080",
			},
			wantNotContains: []string{
				"(allow process-fork)",
			},
		},
	}

	// Test with proxy disabled
	t.Run("proxy disabled - network fully blocked", func(t *testing.T) {
		profile, err := GenerateSeatbeltProfile(
			0, 0, // ports don't matter when proxy is disabled
			false, // enableProxy = false
			[]string{},
			[]string{},
			[]string{},
			[]string{},
			true, true, true, true,
		)

		if err != nil {
			t.Fatalf("GenerateSeatbeltProfile() error = %v", err)
		}

		// Should contain deny network but NOT proxy rules
		if !containsString(profile, "(deny network*)") {
			t.Error("Profile should contain network deny rule")
		}
		if containsString(profile, "localhost:") {
			t.Error("Profile should not contain proxy rules when proxy is disabled")
		}

		// Validate the generated profile
		tmpDir := t.TempDir()
		profilePath := filepath.Join(tmpDir, "no-proxy-profile.sb")
		if err := os.WriteFile(profilePath, []byte(profile), 0600); err != nil {
			t.Fatalf("Failed to write profile: %v", err)
		}

		if err := ValidateProfile(profilePath); err != nil {
			t.Errorf("Profile validation failed: %v", err)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Enable proxy for all tests (testing proxy-enabled mode)
			profile, err := GenerateSeatbeltProfile(
				tt.httpPort,
				tt.socksPort,
				true, // enableProxy
				tt.denyReadPaths,
				tt.allowWritePaths,
				tt.denyWritePaths,
				tt.allowUnlinkPaths,
				tt.allowFork,
				tt.allowSysctlRead,
				tt.allowMachLookup,
				tt.allowPosixShm,
			)

			if err != nil {
				t.Fatalf("GenerateSeatbeltProfile() error = %v", err)
			}

			for _, want := range tt.wantContains {
				if !containsString(profile, want) {
					t.Errorf("GenerateSeatbeltProfile() profile missing %q", want)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if containsString(profile, notWant) {
					t.Errorf("GenerateSeatbeltProfile() profile should not contain %q", notWant)
				}
			}

			// Validate the generated profile
			tmpDir := t.TempDir()
			profilePath := filepath.Join(tmpDir, "generated-profile.sb")
			if err := os.WriteFile(profilePath, []byte(profile), 0600); err != nil {
				t.Fatalf("Failed to write generated profile: %v", err)
			}

			if err := ValidateProfile(profilePath); err != nil {
				t.Errorf("Generated profile failed validation: %v", err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
