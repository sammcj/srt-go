package packagemanager

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/sammcj/srt-go/internal/cache"
)

// DetectPackageManagersCached detects package managers with caching
func DetectPackageManagersCached(verbose bool) []string {
	// Try to load cache
	pathCache, err := cache.Load()
	if err != nil {
		if verbose {
			slog.Debug("Failed to load cache", "error", err)
		}
	}

	// Check if cache is valid (TTL-based only)
	if pathCache != nil && pathCache.IsValid("") {
		if verbose {
			slog.Debug("Using cached package manager paths", "count", len(pathCache.PackageManagerPaths))
		}
		return pathCache.PackageManagerPaths
	}

	// Cache invalid or doesn't exist, detect package managers
	if verbose {
		slog.Debug("Cache invalid or missing, detecting package managers")
	}

	paths := DetectPackageManagers()

	// Save to cache
	newCache := &cache.PathCache{
		PackageManagerPaths: paths,
		Timestamp:           time.Now(),
	}

	if err := newCache.Save(); err != nil {
		if verbose {
			slog.Debug("Failed to save cache", "error", err)
		}
	} else if verbose {
		slog.Debug("Saved package manager paths to cache", "count", len(paths))
	}

	return paths
}

// DetectPackageManagers detects installed package managers and returns their cache/data paths
// that should be allowed for write operations in the sandbox.
func DetectPackageManagers() []string {
	var paths []string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return paths
	}

	// Homebrew (ARM)
	if dirExists("/opt/homebrew") {
		paths = append(paths, "/opt/homebrew/**")
	}

	// Homebrew (Intel)
	if dirExists("/usr/local/Homebrew") {
		paths = append(paths, "/usr/local/Homebrew/**")
	}

	// Nix
	if dirExists("/nix/store") {
		paths = append(paths, "/nix/store/**")
	}
	nixProfile := filepath.Join(homeDir, ".nix-profile")
	if dirExists(nixProfile) {
		paths = append(paths, nixProfile+"/**")
	}

	// Node.js - nvm
	nvmDir := filepath.Join(homeDir, ".nvm")
	if dirExists(nvmDir) {
		paths = append(paths, nvmDir+"/**")
	}

	// Node.js - fnm
	fnmDir := filepath.Join(homeDir, ".fnm")
	if dirExists(fnmDir) {
		paths = append(paths, fnmDir+"/**")
	}

	// Node.js - nodenv
	nodenvDir := filepath.Join(homeDir, ".nodenv")
	if dirExists(nodenvDir) {
		paths = append(paths, nodenvDir+"/**")
	}

	// Deno
	denoDir := filepath.Join(homeDir, ".deno")
	if dirExists(denoDir) {
		paths = append(paths, denoDir+"/**")
	}

	// Bun
	bunDir := filepath.Join(homeDir, ".bun")
	if dirExists(bunDir) {
		paths = append(paths, bunDir+"/**")
	}

	// Python - pyenv
	pyenvDir := filepath.Join(homeDir, ".pyenv")
	if dirExists(pyenvDir) {
		paths = append(paths, pyenvDir+"/**")
	}

	// Python - Poetry
	poetryDir := filepath.Join(homeDir, ".poetry")
	if dirExists(poetryDir) {
		paths = append(paths, poetryDir+"/**")
	}

	// Python - pipx
	pipxDir := filepath.Join(homeDir, ".local", "pipx")
	if dirExists(pipxDir) {
		paths = append(paths, pipxDir+"/**")
	}

	// Python - Conda/Miniconda
	condaDirs := []string{
		filepath.Join(homeDir, "miniconda3"),
		filepath.Join(homeDir, "anaconda3"),
		filepath.Join(homeDir, ".conda"),
	}
	for _, dir := range condaDirs {
		if dirExists(dir) {
			paths = append(paths, dir+"/**")
		}
	}

	// Go - workspace
	goDir := filepath.Join(homeDir, "go")
	if dirExists(goDir) {
		paths = append(paths, goDir+"/**")
	}

	// Go - g version manager
	gDir := filepath.Join(homeDir, ".g")
	if dirExists(gDir) {
		paths = append(paths, gDir+"/**")
	}

	// Java - SDKMAN
	sdkmanDir := filepath.Join(homeDir, ".sdkman")
	if dirExists(sdkmanDir) {
		paths = append(paths, sdkmanDir+"/**")
	}

	// Java - jenv
	jenvDir := filepath.Join(homeDir, ".jenv")
	if dirExists(jenvDir) {
		paths = append(paths, jenvDir+"/**")
	}

	// Ruby - rbenv
	rbenvDir := filepath.Join(homeDir, ".rbenv")
	if dirExists(rbenvDir) {
		paths = append(paths, rbenvDir+"/**")
	}

	// Ruby - RVM
	rvmDir := filepath.Join(homeDir, ".rvm")
	if dirExists(rvmDir) {
		paths = append(paths, rvmDir+"/**")
	}

	// Rust - Cargo
	cargoDir := filepath.Join(homeDir, ".cargo")
	if dirExists(cargoDir) {
		paths = append(paths, cargoDir+"/**")
	}

	// Rust - Rustup
	rustupDir := filepath.Join(homeDir, ".rustup")
	if dirExists(rustupDir) {
		paths = append(paths, rustupDir+"/**")
	}

	// Standard package manager caches (always include these)
	standardCaches := []string{
		filepath.Join(homeDir, ".npm") + "/**",
		filepath.Join(homeDir, ".cache", "pip") + "/**",
		filepath.Join(homeDir, ".cache", "uv") + "/**",
		filepath.Join(homeDir, ".pnpm-store") + "/**",
		filepath.Join(homeDir, ".cache", "yarn") + "/**",
		filepath.Join(homeDir, ".local", "share", "pnpm") + "/**",
	}

	// Only add standard caches if their parent directories exist
	for _, cache := range standardCaches {
		// Extract parent directory (remove "/**" suffix)
		parentDir := cache[:len(cache)-3]
		if dirExists(parentDir) {
			paths = append(paths, cache)
		}
	}

	return paths
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
