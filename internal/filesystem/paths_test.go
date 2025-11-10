package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContainsGlob(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/Users/test/file.txt", false},
		{"*.txt", true},
		{"**/*.go", true},
		{"/path/to/[abc]", true},
		{"/path/{a,b}", true},
		{"normal/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ContainsGlob(tt.path)
			if got != tt.want {
				t.Errorf("ContainsGlob(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestNormalisePath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "srt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "absolute path",
			path:    tmpDir,
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "tilde expansion",
			path:    "~/test",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalisePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalisePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !filepath.IsAbs(got) {
				t.Errorf("NormalisePath() returned non-absolute path: %q", got)
			}
		})
	}
}
