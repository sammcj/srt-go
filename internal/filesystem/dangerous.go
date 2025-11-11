package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DangerousFileDetector finds dangerous files in directories
type DangerousFileDetector struct {
	useRipgrep            bool
	rgCommand             string
	rgArgs                []string
	dangerousFilePatterns []string
	dangerousDirPatterns  []string
}

// NewDangerousFileDetector creates a new detector
func NewDangerousFileDetector(rgCommand string, rgArgs []string, filePatterns, dirPatterns []string) *DangerousFileDetector {
	detector := &DangerousFileDetector{
		rgCommand:             rgCommand,
		rgArgs:                rgArgs,
		dangerousFilePatterns: filePatterns,
		dangerousDirPatterns:  dirPatterns,
	}

	// Check if ripgrep is available
	if _, err := exec.LookPath(rgCommand); err == nil {
		detector.useRipgrep = true
	}

	return detector
}

// Find finds dangerous files in the given root directory
func (d *DangerousFileDetector) Find(root string) ([]string, error) {
	// Normalise root
	normRoot, err := NormalisePath(root)
	if err != nil {
		return nil, err
	}

	// Use ripgrep if available, otherwise walk directory
	if d.useRipgrep {
		return d.findWithRipgrep(normRoot)
	}

	return d.findWithWalk(normRoot)
}

func (d *DangerousFileDetector) findWithRipgrep(root string) ([]string, error) {
	var allMatches []string

	// Search for each pattern
	for _, pattern := range d.dangerousFilePatterns {
		args := append([]string{}, d.rgArgs...)
		args = append(args, "--glob", pattern, root)

		cmd := exec.Command(d.rgCommand, args...)
		output, err := cmd.Output()
		if err != nil {
			// ripgrep returns exit code 1 when no matches found
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				continue
			}
			// Other errors are real errors
			continue
		}

		// Parse output
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line != "" {
				allMatches = append(allMatches, line)
			}
		}
	}

	// Search for dangerous directories
	for _, pattern := range d.dangerousDirPatterns {
		args := append([]string{}, d.rgArgs...)
		args = append(args, "--glob", pattern, root)

		cmd := exec.Command(d.rgCommand, args...)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line != "" {
				// Add the directory path
				dir := filepath.Dir(line)
				allMatches = append(allMatches, filepath.Join(dir, pattern))
			}
		}
	}

	return allMatches, nil
}

func (d *DangerousFileDetector) findWithWalk(root string) ([]string, error) {
	var matches []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		name := entry.Name()

		// Check file patterns
		for _, pattern := range d.dangerousFilePatterns {
			matched, _ := filepath.Match(pattern, name)
			if matched {
				matches = append(matches, path)
				break
			}
		}

		// Check directory patterns
		if entry.IsDir() {
			for _, pattern := range d.dangerousDirPatterns {
				if name == pattern {
					matches = append(matches, path)
					return filepath.SkipDir // Don't descend into this directory
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipDir {
		return nil, fmt.Errorf("directory walk failed: %w", err)
	}

	return matches, nil
}

// GetMandatoryDenyPaths returns dangerous files within allowed write paths
func GetMandatoryDenyPaths(allowWritePaths []string, rgCommand string, rgArgs []string, filePatterns, dirPatterns []string) ([]string, error) {
	detector := NewDangerousFileDetector(rgCommand, rgArgs, filePatterns, dirPatterns)
	var allDangerous []string

	for _, path := range allowWritePaths {
		// Skip if it's a glob pattern
		if ContainsGlob(path) {
			continue
		}

		// Check if path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		dangerous, err := detector.Find(path)
		if err != nil {
			// Don't fail, just skip this path
			continue
		}

		allDangerous = append(allDangerous, dangerous...)
	}

	return allDangerous, nil
}
