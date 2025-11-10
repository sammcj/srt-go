package sandbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// Violation represents a sandbox violation
type Violation struct {
	Process   string    `json:"process"`
	Message   string    `json:"eventMessage"`
	Timestamp time.Time `json:"timestamp"`
	Target    string
	Operation string
}

// ViolationMonitor monitors sandbox violations from system log
type ViolationMonitor struct {
	cmd        *exec.Cmd
	scanner    *bufio.Scanner
	violations chan Violation
	stopCh     chan struct{}
	commandID  string
}

// NewViolationMonitor creates a new violation monitor
func NewViolationMonitor(commandID string) (*ViolationMonitor, error) {
	// Build predicate to filter sandboxd logs containing our command ID
	predicate := fmt.Sprintf(
		"process == 'sandboxd' AND eventMessage CONTAINS '%s'",
		commandID,
	)

	cmd := exec.Command("log", "stream",
		"--predicate", predicate,
		"--style", "json",
		"--level", "default",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start log stream: %w", err)
	}

	return &ViolationMonitor{
		cmd:        cmd,
		scanner:    bufio.NewScanner(stdout),
		violations: make(chan Violation, 100),
		stopCh:     make(chan struct{}),
		commandID:  commandID,
	}, nil
}

// Start begins monitoring violations
func (m *ViolationMonitor) Start() {
	go func() {
		defer close(m.violations)

		for m.scanner.Scan() {
			select {
			case <-m.stopCh:
				return
			default:
				line := m.scanner.Text()

				var v Violation
				if err := json.Unmarshal([]byte(line), &v); err != nil {
					continue
				}

				m.parseViolation(&v)
				m.violations <- v
			}
		}
	}()
}

// Violations returns the violations channel
func (m *ViolationMonitor) Violations() <-chan Violation {
	return m.violations
}

// Stop stops monitoring
func (m *ViolationMonitor) Stop() {
	close(m.stopCh)
	if m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}
}

func (m *ViolationMonitor) parseViolation(v *Violation) {
	// Parse the violation message to extract operation and target
	// Format: "Sandbox: process(pid) deny(1) operation target"
	msg := v.Message

	// Extract operation
	if strings.Contains(msg, "file-read") {
		v.Operation = "file-read"
	} else if strings.Contains(msg, "file-write") {
		v.Operation = "file-write"
	} else if strings.Contains(msg, "network") {
		v.Operation = "network"
	}

	// Extract target (last part of message)
	parts := strings.Fields(msg)
	if len(parts) > 0 {
		v.Target = parts[len(parts)-1]
	}
}

// ShouldIgnoreViolation checks if a violation should be ignored
func ShouldIgnoreViolation(v Violation, ignoreMap map[string][]string) bool {
	// Check for command-specific ignores
	if patterns, ok := ignoreMap[v.Process]; ok {
		for _, pattern := range patterns {
			if strings.Contains(v.Target, pattern) {
				return true
			}
		}
	}

	// Check for global ignores
	if patterns, ok := ignoreMap["*"]; ok {
		for _, pattern := range patterns {
			if strings.Contains(v.Target, pattern) {
				return true
			}
		}
	}

	return false
}

// LogViolation logs a violation
func LogViolation(v Violation) {
	slog.Warn("Sandbox violation",
		"process", v.Process,
		"operation", v.Operation,
		"target", v.Target,
		"time", v.Timestamp,
	)
}
