package sandbox

import (
	"fmt"
	"strings"

	"github.com/sammcj/srt-go/internal/filesystem"
)

// GenerateSeatbeltProfile generates a Seatbelt profile from paths
func GenerateSeatbeltProfile(
	httpProxyPort, socksProxyPort int,
	denyReadPaths, allowWritePaths, denyWritePaths []string,
) (string, error) {
	var sb strings.Builder

	// Version declaration
	sb.WriteString("(version 1)\n\n")

	// Process execution - allow all
	sb.WriteString("; Process execution - allow all\n")
	sb.WriteString("(allow process-exec*)\n\n")

	// Network restrictions - deny all except proxies
	sb.WriteString("; Network - deny all except proxies\n")
	sb.WriteString("(deny network*)\n")
	sb.WriteString(fmt.Sprintf("(allow network* (remote ip \"localhost:%d\"))\n", httpProxyPort))
	sb.WriteString(fmt.Sprintf("(allow network* (remote ip \"localhost:%d\"))\n", socksProxyPort))
	sb.WriteString("\n")

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

	// Prevent file movement bypass
	sb.WriteString("; Prevent bypassing via file movement\n")
	sb.WriteString("(deny file-write-unlink)\n")

	return sb.String(), nil
}
