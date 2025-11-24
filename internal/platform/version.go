package platform

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Version represents a macOS version
type Version struct {
	Major int
	Minor int
	Patch int
	Full  string
}

// String returns the full version string
func (v *Version) String() string {
	return v.Full
}

// GetMacOSVersion retrieves the current macOS version
func GetMacOSVersion() (*Version, error) {
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute sw_vers: %w", err)
	}

	versionStr := strings.TrimSpace(string(output))
	parts := strings.Split(versionStr, ".")

	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid version format: %s", versionStr)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor := 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}

	patch := 0
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
		Full:  versionStr,
	}, nil
}

// CheckSystemRequirements verifies that the system meets requirements
func CheckSystemRequirements() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS required, got: %s", runtime.GOOS)
	}

	version, err := GetMacOSVersion()
	if err != nil {
		return fmt.Errorf("failed to detect macOS version: %w", err)
	}

	if version.Major < 26 {
		return fmt.Errorf("macOS 26 (Tahoe) or newer required, got: %s", version.Full)
	}

	return nil
}

// GetHomeDir returns the user's home directory
func GetHomeDir() (string, error) {
	cmd := exec.Command("sh", "-c", "echo $HOME")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
