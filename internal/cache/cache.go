package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PathCache stores cached path information with TTL
type PathCache struct {
	PackageManagerPaths []string  `json:"packageManagerPaths"`
	ConfigMtime         time.Time `json:"configMtime"`
	Timestamp           time.Time `json:"timestamp"`
}

// DefaultTTL is the default cache TTL (1 hour)
const DefaultTTL = 1 * time.Hour

// GetCachePath returns the path to the cache file
func GetCachePath() (string, error) {
	tmpDir := os.TempDir()
	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}
	return filepath.Join(tmpDir, fmt.Sprintf(".srt-cache-%s.json", username)), nil
}

// Load loads the cache from disk
func Load() (*PathCache, error) {
	cachePath, err := GetCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache doesn't exist, not an error
		}
		return nil, err
	}

	var cache PathCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// Save saves the cache to disk
func (c *PathCache) Save() error {
	cachePath, err := GetCachePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0600)
}

// IsValid checks if the cache is still valid based on TTL and config modification time
func (c *PathCache) IsValid(configPath string) bool {
	if c == nil {
		return false
	}

	// Check TTL from environment or use default
	ttl := DefaultTTL
	if ttlEnv := os.Getenv("SRT_CACHE_TTL"); ttlEnv != "" {
		if d, err := time.ParseDuration(ttlEnv); err == nil {
			ttl = d
		}
	}

	// Check if cache has expired
	if time.Since(c.Timestamp) > ttl {
		return false
	}

	// Check if config file was modified
	if configPath != "" {
		stat, err := os.Stat(configPath)
		if err == nil {
			if stat.ModTime().After(c.ConfigMtime) {
				return false
			}
		}
	}

	return true
}

// GetConfigMtime returns the modification time of the config file
func GetConfigMtime(configPath string) time.Time {
	if configPath == "" {
		return time.Time{}
	}

	stat, err := os.Stat(configPath)
	if err != nil {
		return time.Time{}
	}

	return stat.ModTime()
}

// Clear removes the cache file
func Clear() error {
	cachePath, err := GetCachePath()
	if err != nil {
		return err
	}

	err = os.Remove(cachePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
