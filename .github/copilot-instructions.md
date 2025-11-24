# GitHub Copilot Instructions for srt

## Project Overview

A lightweight, high-performance sandboxing tool for macOS 26+ (Tahoe) that enforces filesystem and network restrictions using the macOS Seatbelt framework.

## Overview

`srt` wraps commands in a security sandbox that controls what files they can access and which network domains they can connect to. It's designed for adding a line of defence when running untrusted code, tools, and other commands that AI agents may run with granular security policies.

## Development Setup

### Building the Project

```bash
# Build the server
make build

# The binary will be created at: bin/srt
```

### Testing

```bash
# Run all tests
make test
```

### Linting and Code Quality

```bash
# Format code
make fmt

# Run linters and modernisation checks
make lint
# This runs: gofmt, golangci-lint, and gopls modernize
```

## Contribution Guidelines

### Before Committing

1. **Format your code:** `make fmt`
2. **Run linters:** `make lint` (must pass without errors)
3. **Run tests:** `make test` (must pass all tests)
4. **Build successfully:** `make build` (must compile without errors)

### Code Standards

- Follow Go best practices and idiomatic patterns
- Use Australian English spelling throughout code and documentation
- No marketing terms like "comprehensive" or "production-grade"
- Focus on clear, concise, actionable technical guidance

## Critical Review Areas

### Go Code Standards
- Follow Go best practices and idiomatic patterns
- Use proper error handling with wrapped errors
- Implement context cancellation correctly
- Ensure goroutine safety and proper synchronisation
- Use appropriate logging with logrus logger
- Follow the project's naming conventions

### Documentation Standards
- Use Australian English spelling throughout
- Provide clear examples and usage patterns
- Document security requirements and limitations
- Documentation should be concise, favouring clear technical information over verbosity

## Code Quality Checks

### General Code Quality
- Verify proper module imports and dependencies
- Check for hardcoded credentials or sensitive data
- Ensure proper resource cleanup (defer statements)
- Validate input parameters thoroughly
- Use appropriate data types and structures
- Follow consistent error message formatting

## Configuration & Environment
- Environment variables should have sensible defaults
- Configuration should be documented in README
- Support both development and production modes
- Handle missing optional dependencies gracefully

## General Guidelines

- Do not use marketing terms such as 'comprehensive' or 'production-grade' in documentation or code comments.
- Focus on clear, concise actionable technical guidance.
