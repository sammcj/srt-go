package filesystem

import (
	"testing"
)

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"*.txt", "^[^/]*\\.txt$"},
		{"**/*.js", "^.*[^/]*\\.js$"}, // ** matches any depth, * matches filename
		{"file?.txt", "^file[^/]\\.txt$"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got, err := GlobToRegex(tt.pattern)
			if err != nil {
				t.Errorf("GlobToRegex() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("GlobToRegex(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.txt", "file.txt", true},
		{"*.txt", "file.go", false},
		{"**/*.js", "src/main.js", true},
		{"**/*.js", "test/unit/helper.js", true},
		{"src/**/*.go", "src/internal/config.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_vs_"+tt.path, func(t *testing.T) {
			got, err := MatchGlob(tt.pattern, tt.path)
			if err != nil {
				t.Errorf("MatchGlob() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("MatchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}
