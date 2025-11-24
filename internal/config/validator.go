package config

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Domain pattern: alphanumeric, hyphens, dots, or wildcard subdomain
	domainPattern = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
)

// Validate checks if the configuration is valid
func Validate(cfg *Config) error {
	// Validate network configuration
	if err := validateNetwork(&cfg.Network); err != nil {
		return fmt.Errorf("network config: %w", err)
	}

	// Validate filesystem configuration
	if err := validateFilesystem(&cfg.Filesystem); err != nil {
		return fmt.Errorf("filesystem config: %w", err)
	}

	return nil
}

func validateNetwork(nc *NetworkConfig) error {
	// Validate allowed domains
	for _, domain := range nc.AllowedDomains {
		if err := validateDomain(domain); err != nil {
			return fmt.Errorf("invalid allowed domain %q: %w", domain, err)
		}
	}

	// Validate denied domains
	for _, domain := range nc.DeniedDomains {
		if err := validateDomain(domain); err != nil {
			return fmt.Errorf("invalid denied domain %q: %w", domain, err)
		}
	}

	// Validate ports
	if nc.HTTPProxyPort < 0 || nc.HTTPProxyPort > 65535 {
		return fmt.Errorf("invalid HTTP proxy port: %d", nc.HTTPProxyPort)
	}

	if nc.SOCKSProxyPort < 0 || nc.SOCKSProxyPort > 65535 {
		return fmt.Errorf("invalid SOCKS proxy port: %d", nc.SOCKSProxyPort)
	}

	return nil
}

func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Reject overly broad patterns
	if domain == "*" || domain == "*.*" {
		return fmt.Errorf("overly broad pattern")
	}

	// Check for TLD-only wildcards (*.com, *.org)
	if strings.HasPrefix(domain, "*.") {
		parts := strings.Split(domain[2:], ".")
		if len(parts) == 1 {
			return fmt.Errorf("TLD-only wildcards not allowed")
		}
	}

	// Validate format
	if !domainPattern.MatchString(domain) {
		return fmt.Errorf("invalid domain format")
	}

	return nil
}

func validateFilesystem(fc *FilesystemConfig) error {
	// Validate deny read paths
	for _, path := range fc.DenyRead {
		if path == "" {
			return fmt.Errorf("deny read path cannot be empty")
		}
	}

	// Validate allow write paths
	for _, path := range fc.AllowWrite {
		if path == "" {
			return fmt.Errorf("allow write path cannot be empty")
		}
	}

	// Validate deny write paths
	for _, path := range fc.DenyWrite {
		if path == "" {
			return fmt.Errorf("deny write path cannot be empty")
		}
	}

	return nil
}
