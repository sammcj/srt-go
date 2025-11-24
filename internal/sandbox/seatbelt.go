package sandbox

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sammcj/srt-go/internal/filesystem"
)

// GenerateSeatbeltProfile generates a Seatbelt profile from paths and process permissions
func GenerateSeatbeltProfile(
	httpProxyPort, socksProxyPort int,
	enableProxy bool,
	denyReadPaths, allowWritePaths, denyWritePaths, allowUnlinkPaths []string,
	allowFork, allowSysctlRead, allowMachLookup, allowPosixShm bool,
) (string, error) {
	var sb strings.Builder

	// Version declaration
	sb.WriteString("(version 1)\n\n")

	// Process operations - configurable permissions
	sb.WriteString("; Process operations\n")
	sb.WriteString("(allow process-exec*)\n")
	if allowFork {
		sb.WriteString("(allow process-fork)\n")
	}
	if allowSysctlRead {
		sb.WriteString("(allow sysctl-read)\n")
	}
	if allowMachLookup {
		sb.WriteString("(allow mach-lookup)\n")
	}
	if allowPosixShm {
		sb.WriteString("(allow ipc-posix-shm*)\n")
	}
	sb.WriteString("\n")

	// Network restrictions
	if enableProxy {
		// Deny all except proxies
		sb.WriteString("; Network - deny all except proxies\n")
		sb.WriteString("(deny network*)\n")
		sb.WriteString(fmt.Sprintf("(allow network* (remote ip \"localhost:%d\"))\n", httpProxyPort))
		sb.WriteString(fmt.Sprintf("(allow network* (remote ip \"localhost:%d\"))\n", socksProxyPort))
		sb.WriteString("\n")
	} else {
		// Deny all network access
		sb.WriteString("; Network - deny all\n")
		sb.WriteString("(deny network*)\n\n")
	}

	// File reads - allow by default, deny specific
	sb.WriteString("; Filesystem reads - allow by default\n")
	sb.WriteString("(allow file-read*)\n\n")

	if len(denyReadPaths) > 0 {
		sb.WriteString("; Deny specific read paths\n")
		for _, path := range denyReadPaths {
			if filesystem.ContainsGlob(path) {
				regex, err := filesystem.GlobToRegex(path)
				if err != nil {
					return "", fmt.Errorf("failed to convert glob %q: %w", path, err)
				}
				sb.WriteString(fmt.Sprintf("(deny file-read* (regex #\"%s\"))\n", regex))
			} else {
				sb.WriteString(fmt.Sprintf("(deny file-read* (subpath \"%s\"))\n", path))
			}
		}
		sb.WriteString("\n")
	}

	// File writes - deny by default, allow specific
	sb.WriteString("; Filesystem writes - deny by default\n")
	sb.WriteString("(deny file-write*)\n\n")

	if len(allowWritePaths) > 0 {
		sb.WriteString("; Allow writes to specific paths\n")
		for _, path := range allowWritePaths {
			if filesystem.ContainsGlob(path) {
				regex, err := filesystem.GlobToRegex(path)
				if err != nil {
					return "", fmt.Errorf("failed to convert glob %q: %w", path, err)
				}
				sb.WriteString(fmt.Sprintf("(allow file-write* (regex #\"%s\"))\n", regex))
			} else {
				sb.WriteString(fmt.Sprintf("(allow file-write* (subpath \"%s\"))\n", path))
			}
		}
		sb.WriteString("\n")
	}

	// Deny writes within allowed paths
	if len(denyWritePaths) > 0 {
		sb.WriteString("; Deny specific writes within allowed paths\n")
		for _, path := range denyWritePaths {
			if filesystem.ContainsGlob(path) {
				regex, err := filesystem.GlobToRegex(path)
				if err != nil {
					return "", fmt.Errorf("failed to convert glob %q: %w", path, err)
				}
				sb.WriteString(fmt.Sprintf("(deny file-write* (regex #\"%s\"))\n", regex))
			} else {
				sb.WriteString(fmt.Sprintf("(deny file-write* (subpath \"%s\"))\n", path))
			}
		}
		sb.WriteString("\n")
	}

	// File unlink/deletion - deny by default, allow specific
	sb.WriteString("; File unlink/deletion - deny by default\n")
	sb.WriteString("(deny file-write-unlink)\n\n")

	if len(allowUnlinkPaths) > 0 {
		sb.WriteString("; Allow unlink in specific paths\n")
		for _, path := range allowUnlinkPaths {
			if filesystem.ContainsGlob(path) {
				regex, err := filesystem.GlobToRegex(path)
				if err != nil {
					return "", fmt.Errorf("failed to convert glob %q: %w", path, err)
				}
				sb.WriteString(fmt.Sprintf("(allow file-write-unlink (regex #\"%s\"))\n", regex))
			} else {
				sb.WriteString(fmt.Sprintf("(allow file-write-unlink (subpath \"%s\"))\n", path))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// ValidateProfile validates a Seatbelt profile both syntactically and by live testing
func ValidateProfile(profilePath string) error {
	// Phase 1: Syntax validation
	content, err := exec.Command("cat", profilePath).Output()
	if err != nil {
		return fmt.Errorf("failed to read profile: %w", err)
	}

	profileStr := string(content)

	// Check for version declaration
	if !strings.Contains(profileStr, "(version 1)") {
		return fmt.Errorf("profile missing (version 1) declaration")
	}

	// Check for balanced parentheses
	if !hasBalancedParentheses(profileStr) {
		return fmt.Errorf("profile has unbalanced parentheses")
	}

	// Check for at least one deny or allow statement
	hasDeny := strings.Contains(profileStr, "(deny ")
	hasAllow := strings.Contains(profileStr, "(allow ")
	if !hasDeny && !hasAllow {
		return fmt.Errorf("profile missing deny/allow statements")
	}

	// Phase 2: Live testing
	cmd := exec.Command("sandbox-exec", "-f", profilePath, "/usr/bin/true")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("profile validation failed (sandbox-exec test): %w\nOutput: %s", err, string(output))
	}

	return nil
}

// hasBalancedParentheses checks if parentheses are balanced in the profile
func hasBalancedParentheses(s string) bool {
	count := 0
	for _, ch := range s {
		switch ch {
		case '(':
			count++
		case ')':
			count--
			if count < 0 {
				return false
			}
		}
	}
	return count == 0
}
