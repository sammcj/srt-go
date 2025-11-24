package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalisePath normalises a path by expanding ~, resolving symlinks, and making it absolute
func NormalisePath(path string) (string, error) {
	// Expand tilde
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Make absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to make path absolute: %w", err)
	}

	// Resolve symlinks if path exists
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If path doesn't exist, just return the absolute path
		if os.IsNotExist(err) {
			return absPath, nil
		}
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	return resolvedPath, nil
}

// ContainsGlob checks if a path contains glob characters
func ContainsGlob(path string) bool {
	return strings.ContainsAny(path, "*?[{")
}

// NormalisePaths normalises a slice of paths
func NormalisePaths(paths []string) ([]string, error) {
	normalised := make([]string, 0, len(paths))

	for _, path := range paths {
		// Expand tilde even for glob patterns
		if strings.HasPrefix(path, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			path = filepath.Join(home, path[1:])
		}

		// For glob patterns, don't resolve symlinks or make absolute
		// Just expand tilde and keep as-is
		if ContainsGlob(path) {
			normalised = append(normalised, path)
			continue
		}

		// For non-glob paths, do full normalisation
		normPath, err := NormalisePath(path)
		if err != nil {
			return nil, fmt.Errorf("failed to normalise %q: %w", path, err)
		}

		normalised = append(normalised, normPath)
	}

	return normalised, nil
}
