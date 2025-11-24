package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// ViolationLogger handles logging violations to a rotating file
type ViolationLogger struct {
	logger *log.Logger
	file   *lumberjack.Logger
}

// NewViolationLogger creates a new violation logger
func NewViolationLogger() (*ViolationLogger, error) {
	// Determine log file path
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	logDir := filepath.Join(home, ".srt")
	logPath := filepath.Join(logDir, "deny.log")

	// Ensure directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Configure rotating file logger
	rotatingFile := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    512, // kilobytes (512KB as requested)
		MaxBackups: 3,   // keep 3 old log files
		MaxAge:     0,   // don't delete based on age
		Compress:   false,
	}

	// Create logger with the rotating file as output
	logger := log.New(rotatingFile, "", log.LstdFlags)

	return &ViolationLogger{
		logger: logger,
		file:   rotatingFile,
	}, nil
}

// LogViolation logs a violation to the file
func (vl *ViolationLogger) LogViolation(v Violation) {
	vl.logger.Printf("VIOLATION process=%s operation=%s target=%s time=%s",
		v.Process,
		v.Operation,
		v.Target,
		v.Timestamp.Format("2006-01-02 15:04:05"),
	)
}

// Close closes the log file
func (vl *ViolationLogger) Close() error {
	if vl.file != nil {
		return vl.file.Close()
	}
	return nil
}
