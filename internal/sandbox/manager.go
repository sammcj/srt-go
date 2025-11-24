package sandbox

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/sammcj/srt-go/internal/config"
	"github.com/sammcj/srt-go/internal/filesystem"
	"github.com/sammcj/srt-go/internal/network"
	"github.com/sammcj/srt-go/internal/packagemanager"
)

// Manager orchestrates sandbox execution
type Manager struct {
	config          *config.Config
	httpProxy       *network.HTTPProxy
	socksProxy      *network.SOCKSProxy
	profilePath     string
	violationMon    *ViolationMonitor
	violationLogger *ViolationLogger
	commandID       string
	wg              sync.WaitGroup
	stopCh          chan struct{}
}

// NewManager creates a new sandbox manager
func NewManager(cfg *config.Config) (*Manager, error) {
	mgr := &Manager{
		config:    cfg,
		stopCh:    make(chan struct{}),
		commandID: generateCommandID(),
	}

	// Create violation logger (always created, logs all violations to file)
	violationLogger, err := NewViolationLogger()
	if err != nil {
		// Don't fail if we can't create the logger, just warn
		slog.Debug("Failed to create violation logger", "error", err)
	} else {
		mgr.violationLogger = violationLogger
	}

	// Determine if proxy is needed based on network configuration
	needsProxy := needsNetworkProxy(cfg)

	if needsProxy {
		// Create domain filter
		filter, err := network.NewDomainFilter(
			cfg.Network.DefaultPolicy,
			cfg.Network.AllowedDomains,
			cfg.Network.DeniedDomains,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create domain filter: %w", err)
		}

		// Create HTTP proxy
		mgr.httpProxy, err = network.NewHTTPProxy(filter, cfg.Network.HTTPProxyPort)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP proxy: %w", err)
		}

		// Update config with actual port
		cfg.Network.HTTPProxyPort = mgr.httpProxy.Port()

		// Create SOCKS5 proxy
		mgr.socksProxy, err = network.NewSOCKSProxy(filter, cfg.Network.SOCKSProxyPort)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 proxy: %w", err)
		}

		// Update config with actual port
		cfg.Network.SOCKSProxyPort = mgr.socksProxy.Port()

		// Start proxies in background
		mgr.wg.Add(2)

		go func() {
			defer mgr.wg.Done()
			if err := mgr.httpProxy.Start(); err != nil {
				slog.Debug("HTTP proxy stopped", "error", err)
			}
		}()

		go func() {
			defer mgr.wg.Done()
			if err := mgr.socksProxy.Start(); err != nil {
				slog.Debug("SOCKS5 proxy stopped", "error", err)
			}
		}()

		if cfg.Verbose {
			slog.Debug("Network proxies started",
				"http_port", cfg.Network.HTTPProxyPort,
				"socks_port", cfg.Network.SOCKSProxyPort)
		}
	} else {
		if cfg.Verbose {
			slog.Debug("Network proxies not needed - all network access blocked")
		}
	}

	// Set up cleanup on signals
	mgr.setupCleanup()

	return mgr, nil
}

// needsNetworkProxy determines if network proxies are needed based on configuration
func needsNetworkProxy(cfg *config.Config) bool {
	// Proxy needed if we have allowed domains (filtering mode)
	if len(cfg.Network.AllowedDomains) > 0 {
		return true
	}

	// Proxy needed if default policy is "allow" (deny-list mode)
	if cfg.Network.DefaultPolicy == "allow" {
		return true
	}

	// Otherwise, no proxy needed (all network blocked)
	return false
}

// DryRun shows what would be executed without actually running the command
func (m *Manager) DryRun(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}

	fmt.Println("[srt-go] Dry-run mode enabled")
	fmt.Println()

	// Detect package managers and add their paths to allowWrite (with caching)
	detectedPaths := packagemanager.DetectPackageManagersCached(m.config.Verbose)
	if len(detectedPaths) > 0 {
		fmt.Printf("[srt-go] Detected package manager paths: %d paths\n", len(detectedPaths))
		m.config.Filesystem.AllowWrite = append(m.config.Filesystem.AllowWrite, detectedPaths...)
		m.config.Filesystem.AllowUnlink = append(m.config.Filesystem.AllowUnlink, detectedPaths...)
	}

	// Normalise filesystem paths
	denyReadPaths, err := filesystem.NormalisePaths(m.config.Filesystem.DenyRead)
	if err != nil {
		return fmt.Errorf("failed to normalise deny read paths: %w", err)
	}

	allowWritePaths, err := filesystem.NormalisePaths(m.config.Filesystem.AllowWrite)
	if err != nil {
		return fmt.Errorf("failed to normalise allow write paths: %w", err)
	}

	denyWritePaths, err := filesystem.NormalisePaths(m.config.Filesystem.DenyWrite)
	if err != nil {
		return fmt.Errorf("failed to normalise deny write paths: %w", err)
	}

	allowUnlinkPaths, err := filesystem.NormalisePaths(m.config.Filesystem.AllowUnlink)
	if err != nil {
		return fmt.Errorf("failed to normalise allow unlink paths: %w", err)
	}

	// Get mandatory deny paths (dangerous files in allowed write dirs)
	mandatoryDeny, err := filesystem.GetMandatoryDenyPaths(
		allowWritePaths,
		m.config.Ripgrep.Command,
		m.config.Ripgrep.Args,
		m.config.ScanAndBlockFiles,
		m.config.ScanAndBlockDirs,
	)
	if err == nil {
		denyWritePaths = append(denyWritePaths, mandatoryDeny...)
		if len(mandatoryDeny) > 0 {
			fmt.Printf("[srt-go] Found %d dangerous files/directories in write-allowed paths\n", len(mandatoryDeny))
		}
	}

	// Determine if proxies are enabled
	proxyEnabled := m.httpProxy != nil && m.socksProxy != nil

	// Generate Seatbelt profile
	profile, err := GenerateSeatbeltProfile(
		m.config.Network.HTTPProxyPort,
		m.config.Network.SOCKSProxyPort,
		proxyEnabled,
		denyReadPaths,
		allowWritePaths,
		denyWritePaths,
		allowUnlinkPaths,
		m.config.Process.AllowFork,
		m.config.Process.AllowSysctlRead,
		m.config.Process.AllowMachLookup,
		m.config.Process.AllowPosixShm,
	)
	if err != nil {
		return fmt.Errorf("failed to generate Seatbelt profile: %w", err)
	}

	// Print profile
	fmt.Println("[srt-go] Generated Seatbelt profile:")
	fmt.Println()
	fmt.Println(profile)
	fmt.Println()

	// Build sandbox-exec command
	args := []string{"-f", "<profile-file>"}
	args = append(args, command...)

	fmt.Println("[srt-go] Would execute:")
	fmt.Printf("  sandbox-exec %s\n", strings.Join(args, " "))
	fmt.Println()

	// Show environment variables
	fmt.Println("[srt-go] Environment variables:")
	fmt.Printf("  SRT_COMMAND_ID=%s\n", m.commandID)
	if proxyEnabled {
		fmt.Printf("  HTTP_PROXY=http://localhost:%d\n", m.config.Network.HTTPProxyPort)
		fmt.Printf("  HTTPS_PROXY=http://localhost:%d\n", m.config.Network.HTTPProxyPort)
		fmt.Printf("  ALL_PROXY=socks5://localhost:%d\n", m.config.Network.SOCKSProxyPort)
	} else {
		fmt.Println("  (No proxy environment variables - network fully blocked)")
	}
	fmt.Println()

	// Show filesystem permissions summary
	fmt.Println("[srt-go] Filesystem permissions:")
	fmt.Printf("  Deny read: %d paths\n", len(denyReadPaths))
	fmt.Printf("  Allow write: %d paths\n", len(allowWritePaths))
	fmt.Printf("  Deny write: %d paths\n", len(denyWritePaths))
	fmt.Printf("  Allow unlink: %d paths\n", len(allowUnlinkPaths))
	fmt.Println()

	// Show network configuration
	fmt.Println("[srt-go] Network configuration:")
	fmt.Printf("  Default policy: %s\n", m.config.Network.DefaultPolicy)
	fmt.Printf("  Allowed domains: %d\n", len(m.config.Network.AllowedDomains))
	fmt.Printf("  Denied domains: %d\n", len(m.config.Network.DeniedDomains))
	fmt.Printf("  Proxy enabled: %v\n", proxyEnabled)
	fmt.Println()

	return nil
}

// Execute runs a command in the sandbox
func (m *Manager) Execute(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}

	// Detect package managers and add their paths to allowWrite (with caching)
	detectedPaths := packagemanager.DetectPackageManagersCached(m.config.Verbose)
	if len(detectedPaths) > 0 {
		if m.config.Verbose {
			slog.Debug("Detected package manager paths", "count", len(detectedPaths), "paths", detectedPaths)
		}
		m.config.Filesystem.AllowWrite = append(m.config.Filesystem.AllowWrite, detectedPaths...)
		m.config.Filesystem.AllowUnlink = append(m.config.Filesystem.AllowUnlink, detectedPaths...)
	}

	// Normalise filesystem paths
	denyReadPaths, err := filesystem.NormalisePaths(m.config.Filesystem.DenyRead)
	if err != nil {
		return fmt.Errorf("failed to normalise deny read paths: %w", err)
	}

	allowWritePaths, err := filesystem.NormalisePaths(m.config.Filesystem.AllowWrite)
	if err != nil {
		return fmt.Errorf("failed to normalise allow write paths: %w", err)
	}

	denyWritePaths, err := filesystem.NormalisePaths(m.config.Filesystem.DenyWrite)
	if err != nil {
		return fmt.Errorf("failed to normalise deny write paths: %w", err)
	}

	allowUnlinkPaths, err := filesystem.NormalisePaths(m.config.Filesystem.AllowUnlink)
	if err != nil {
		return fmt.Errorf("failed to normalise allow unlink paths: %w", err)
	}

	// Get mandatory deny paths (dangerous files in allowed write dirs)
	mandatoryDeny, err := filesystem.GetMandatoryDenyPaths(
		allowWritePaths,
		m.config.Ripgrep.Command,
		m.config.Ripgrep.Args,
		m.config.ScanAndBlockFiles,
		m.config.ScanAndBlockDirs,
	)
	if err != nil {
		slog.Debug("Failed to get mandatory deny paths", "error", err)
		// Don't fail, just continue without them
	} else {
		denyWritePaths = append(denyWritePaths, mandatoryDeny...)
	}

	// Determine if proxies are enabled
	proxyEnabled := m.httpProxy != nil && m.socksProxy != nil

	// Generate Seatbelt profile
	profile, err := GenerateSeatbeltProfile(
		m.config.Network.HTTPProxyPort,
		m.config.Network.SOCKSProxyPort,
		proxyEnabled,
		denyReadPaths,
		allowWritePaths,
		denyWritePaths,
		allowUnlinkPaths,
		m.config.Process.AllowFork,
		m.config.Process.AllowSysctlRead,
		m.config.Process.AllowMachLookup,
		m.config.Process.AllowPosixShm,
	)
	if err != nil {
		return fmt.Errorf("failed to generate Seatbelt profile: %w", err)
	}

	// Write profile to temporary file
	m.profilePath = filepath.Join(os.TempDir(), fmt.Sprintf("srt-profile-%d.sb", os.Getpid()))
	if err := os.WriteFile(m.profilePath, []byte(profile), 0600); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	if m.config.Verbose {
		slog.Info("Generated Seatbelt profile", "path", m.profilePath)
		slog.Debug("Profile content", "profile", profile)
	}

	// Validate the generated profile
	if err := ValidateProfile(m.profilePath); err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	if m.config.Verbose {
		slog.Debug("Seatbelt profile validation passed")
	}

	// Start violation monitoring (always monitor, not just in verbose mode)
	mon, err := NewViolationMonitor(m.commandID)
	if err != nil {
		slog.Debug("Failed to start violation monitor", "error", err)
	} else {
		m.violationMon = mon
		m.violationMon.Start()

		// Process violations in background
		go func() {
			for v := range m.violationMon.Violations() {
				if !ShouldIgnoreViolation(v, m.config.Violations) {
					// Always log to file if logger is available
					if m.violationLogger != nil {
						m.violationLogger.LogViolation(v)
					}
					// Also log to stderr if verbose
					if m.config.Verbose {
						LogViolation(v)
					}
				}
			}
		}()
	}

	// Build sandbox-exec command
	args := []string{"-f", m.profilePath}
	args = append(args, command...)

	cmd := exec.Command("sandbox-exec", args...)

	// Set environment variables
	cmd.Env = append(os.Environ(), fmt.Sprintf("SRT_COMMAND_ID=%s", m.commandID))

	// Set proxy environment variables only if proxies are enabled
	if proxyEnabled {
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("HTTP_PROXY=http://localhost:%d", m.config.Network.HTTPProxyPort),
			fmt.Sprintf("HTTPS_PROXY=http://localhost:%d", m.config.Network.HTTPProxyPort),
			fmt.Sprintf("ALL_PROXY=socks5://localhost:%d", m.config.Network.SOCKSProxyPort),
		)
	}

	// Inherit stdio for interactive commands
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if m.config.Verbose {
		if proxyEnabled {
			slog.Info("Executing sandboxed command",
				"command", strings.Join(command, " "),
				"http_proxy", m.config.Network.HTTPProxyPort,
				"socks_proxy", m.config.Network.SOCKSProxyPort,
			)
		} else {
			slog.Info("Executing sandboxed command",
				"command", strings.Join(command, " "),
				"network", "fully blocked",
			)
		}
	}

	// Execute and wait
	err = cmd.Run()

	// Return exit code if command failed
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

// Cleanup cleans up resources
func (m *Manager) Cleanup() {
	close(m.stopCh)

	// Stop violation monitoring
	if m.violationMon != nil {
		m.violationMon.Stop()
	}

	// Close violation logger
	if m.violationLogger != nil {
		m.violationLogger.Close()
	}

	// Stop proxies
	if m.httpProxy != nil {
		m.httpProxy.Stop()
	}
	if m.socksProxy != nil {
		m.socksProxy.Stop()
	}

	// Wait for goroutines
	m.wg.Wait()

	// Remove profile file
	if m.profilePath != "" {
		os.Remove(m.profilePath)
	}
}

func (m *Manager) setupCleanup() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			m.Cleanup()
			os.Exit(130) // Standard exit code for SIGINT
		case <-m.stopCh:
			return
		}
	}()
}

func generateCommandID() string {
	// Generate a unique ID for this command execution
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("srt-%d", os.Getpid())))
}
