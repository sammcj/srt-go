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
)

// Manager orchestrates sandbox execution
type Manager struct {
	config         *config.Config
	httpProxy      *network.HTTPProxy
	socksProxy     *network.SOCKSProxy
	profilePath    string
	violationMon   *ViolationMonitor
	commandID      string
	wg             sync.WaitGroup
	stopCh         chan struct{}
}

// NewManager creates a new sandbox manager
func NewManager(cfg *config.Config) (*Manager, error) {
	mgr := &Manager{
		config:    cfg,
		stopCh:    make(chan struct{}),
		commandID: generateCommandID(),
	}

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

	// Set up cleanup on signals
	mgr.setupCleanup()

	return mgr, nil
}

// Execute runs a command in the sandbox
func (m *Manager) Execute(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
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

	// Get mandatory deny paths (dangerous files in allowed write dirs)
	mandatoryDeny, err := filesystem.GetMandatoryDenyPaths(
		allowWritePaths,
		m.config.Ripgrep.Command,
		m.config.Ripgrep.Args,
		m.config.DangerousFilePatterns,
		m.config.DangerousDirPatterns,
	)
	if err != nil {
		slog.Debug("Failed to get mandatory deny paths", "error", err)
		// Don't fail, just continue without them
	} else {
		denyWritePaths = append(denyWritePaths, mandatoryDeny...)
	}

	// Generate Seatbelt profile
	profile, err := GenerateSeatbeltProfile(
		m.config.Network.HTTPProxyPort,
		m.config.Network.SOCKSProxyPort,
		denyReadPaths,
		allowWritePaths,
		denyWritePaths,
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

	// Start violation monitoring if verbose
	if m.config.Verbose {
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
						LogViolation(v)
					}
				}
			}()
		}
	}

	// Build sandbox-exec command
	args := []string{"-f", m.profilePath}
	args = append(args, command...)

	cmd := exec.Command("sandbox-exec", args...)

	// Set proxy environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HTTP_PROXY=http://localhost:%d", m.config.Network.HTTPProxyPort),
		fmt.Sprintf("HTTPS_PROXY=http://localhost:%d", m.config.Network.HTTPProxyPort),
		fmt.Sprintf("ALL_PROXY=socks5://localhost:%d", m.config.Network.SOCKSProxyPort),
		fmt.Sprintf("SRT_COMMAND_ID=%s", m.commandID),
	)

	// Inherit stdio for interactive commands
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if m.config.Verbose {
		slog.Info("Executing sandboxed command",
			"command", strings.Join(command, " "),
			"http_proxy", m.config.Network.HTTPProxyPort,
			"socks_proxy", m.config.Network.SOCKSProxyPort,
		)
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
