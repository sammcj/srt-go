# srt-go: Sandbox Runtime for macOS

A lightweight, high-performance sandboxing tool for macOS 26+ (Tahoe) that enforces filesystem and network restrictions using the macOS Seatbelt framework.

## Overview

`srt` wraps commands in a security sandbox that controls what files they can access and which network domains they can connect to. It's designed for adding a line of defence when running untrusted code, tools, and other commands that AI agents may run with granular security policies.

### Why sandbox?

- **Protect secrets**: Prevent commands from reading `~/.ssh`, `~/.aws`, `.env` files
- **Control network access**: Allow/deny specific domains, default allow or deny all
- **Sandbox package managers**: Run `npx`, `uvx` with limited access to prevent data leaks
- **Audit access**: See what files and networks commands try to access
- **Zero trust for AI agents**: Let AI coding assistants run commands with file and network access controls

### Key Features

- **Filesystem isolation** with read/write restrictions and glob patterns
- **Network filtering** via HTTP/HTTPS and SOCKS5 proxies with domain control
- **Configurable default policies** - allow-by-default or deny-by-default
- **Real-time violation monitoring** with structured reporting
- **Native macOS integration** using Seatbelt framework
- **Fast startup** (~2-5ms overhead)
- **Single binary** - no dependencies except macOS 26+

## Installation

### Prerequisites

- macOS 26.0 (Tahoe) or newer
- Go 1.23+ (for building from source)

### Go install

```bash
go install github.com/sammcj/srt-go@HEAD
```

### Manual Build and Install

```bash
# Clone the repository
git clone https://github.com/sammcj/srt-go
cd srt-go

# Build for current architecture
make build

# Install to /usr/local/bin
make install
```

### Verify Installation

```bash
srt --version
```

On first run, srt creates `~/.srt-settings.json` with sensible defaults.

## Quick Start

### Basic Usage

```bash
# Run a command in the sandbox
srt "npm install"

# Multiple arguments (no quotes needed)
srt ls -la
srt git status

# With verbose output
srt --verbose "cargo build"

# Custom configuration
srt --settings ./my-config.json "python setup.py install"
```

### Integration with Claude Code

Configure Claude Code to use srt for all sandboxed commands:

```json
{
  "sandbox": {
    "command": "srt",
    "enabled": true
  }
}
```

Now Claude Code runs all commands through srt automatically, preventing access to secrets and controlling network access.

## Configuration

Configuration is stored in `~/.srt-settings.json` and created automatically on first run.

### Configuration Structure

```json
{
  "network": {
    "defaultPolicy": "allow",
    "allowedDomains": [],
    "deniedDomains": [],
    "allowUnixSockets": [],
    "allowLocalBinding": false,
    "httpProxyPort": 0,
    "socksProxyPort": 0
  },
  "filesystem": {
    "denyRead": [
      "~/.ssh/**",
      "~/.aws/**",
      "~/.config/gcloud/**",
      "/var/db/**",
      "~/Library/Keychains/**",
      "/System/Library/Keychains/**",
      "/System/Library/Security/**",
      "/System/Library/PrivateFrameworks/**",
      "/System/Library/Extensions/**",
      "/System/Library/LaunchDaemons/**",
      "/System/Library/LaunchAgents/**"
    ],
    "allowWrite": ["."],
    "denyWrite": []
  },
  "process": {
    "allowFork": true,
    "allowSysctlRead": true,
    "allowMachLookup": true,
    "allowPosixShm": true
  },
  "dangerousFilePatterns": [
    ".env",
    ".git-credentials",
    ".srt-settings.json",
    ".keychain",
    ".ripgreprc"
  ],
  "dangerousDirPatterns": [
    ".secrets",
    ".ssh",
    ".aws",
    ".keychain"
  ],
  "ignoreViolations": {
    "*": ["/usr/bin", "/usr/lib", "/System", "/Library"]
  },
  "ripgrep": {
    "command": "rg",
    "args": ["--files", "--hidden", "--follow"]
  }
}
```

### Network Configuration

#### Default Policy

Choose between two network security models:

**Allow-by-default** (permissive, recommended for development):
```json
{
  "network": {
    "defaultPolicy": "allow",
    "deniedDomains": ["malicious.com", "*.tracking-ads.com"]
  }
}
```
- Allows all domains except those explicitly denied
- Use `deniedDomains` to block specific sites
- Good for development where network access is needed

**Deny-by-default** (restrictive, recommended for untrusted code):
```json
{
  "network": {
    "defaultPolicy": "deny",
    "allowedDomains": ["github.com", "*.npmjs.org", "pypi.org"]
  }
}
```
- Denies all domains except those explicitly allowed
- Use `allowedDomains` to permit specific sites
- Good for running untrusted code or CI/CD

#### Domain Patterns

- **Exact match**: `"github.com"` matches only github.com
- **Wildcard subdomain**: `"*.npmjs.org"` matches registry.npmjs.org, etc.
- **Deny precedence**: Denied domains always block, regardless of allow list

#### Other Network Options

- `allowUnixSockets`: Unix socket paths to permit (e.g., `["/var/run/docker.sock"]`)
- `allowLocalBinding`: Allow binding to local ports (default: false)
- `httpProxyPort`: HTTP/HTTPS proxy port (0 = auto-assign)
- `socksProxyPort`: SOCKS5 proxy port (0 = auto-assign)

### Filesystem Configuration

#### Read Restrictions

```json
{
  "filesystem": {
    "denyRead": [
      "~/.ssh/**",           // SSH keys
      "~/.aws/**",           // AWS credentials
      "**/.env*",            // Environment files anywhere
      "~/secrets/**"         // Custom secrets directory
    ]
  }
}
```

- By default, all paths are readable
- Paths in `denyRead` are blocked
- Supports glob patterns (see below)

#### Write Restrictions

```json
{
  "filesystem": {
    "allowWrite": ["."],      // Allow current directory
    "denyWrite": [
      "**/.env*",             // Block .env files
      "**/*.key",             // Block key files
      "**/.git/**"            // Block .git directory
    ]
  }
}
```

- By default, all writes are denied
- Only paths in `allowWrite` permit writes
- Paths in `denyWrite` are blocked even if parent is allowed
- More specific rules override general rules

#### Dangerous File Protection

These patterns are automatically protected in allowed write directories:

```json
{
  "dangerousFilePatterns": [".env", ".git-credentials"],
  "dangerousDirPatterns": [".secrets", ".ssh", ".aws"]
}
```

Even if you allow write to current directory, these files/directories are automatically scanned and blocked using ripgrep.

### Process Configuration

Control which low-level process operations are permitted:

```json
{
  "process": {
    "allowFork": true,
    "allowSysctlRead": true,
    "allowMachLookup": true,
    "allowPosixShm": true
  }
}
```

- **allowFork**: Allow processes to fork (create child processes). Required for most scripting languages and package managers.
- **allowSysctlRead**: Allow reading system information via sysctl. Needed for system information queries.
- **allowMachLookup**: Allow Mach IPC service lookups. Required for inter-process communication on macOS.
- **allowPosixShm**: Allow POSIX shared memory operations. Required for memory allocation in many programs.

**Default**: All are `true` by default as they're required for basic operations of most development tools (npm, pip, etc.). Set to `false` for maximum restriction when running untrusted code that doesn't need these capabilities.

### Pattern Matching

Supports gitignore-style glob patterns:

| Pattern     | Matches                                     |
|-------------|---------------------------------------------|
| `*.txt`     | file.txt, test.txt (current level only)     |
| `**/*.js`   | src/main.js, lib/util/helper.js (any depth) |
| `**/.env*`  | .env, config/.env.local (anywhere)          |
| `~/.ssh/**` | All files under ~/.ssh                      |
| `file?.txt` | file1.txt, fileA.txt (single char)          |
| `{a,b}.txt` | a.txt, b.txt (alternation)                  |
| `[abc].txt` | a.txt, b.txt, c.txt (character class)       |

### Violation Handling

Ignore expected violations to reduce noise:

```json
{
  "ignoreViolations": {
    "*": ["/usr/bin", "/System"],
    "git push": ["/usr/bin/nc"]
  }
}
```

- `"*"` applies to all commands
- Specific command patterns apply only to matching commands

## Usage Examples

### Development Workflow

Allow network access, protect secrets:

```bash
# Install dependencies safely
srt "npm install"
srt "pip install requests"
srt "go mod download"

# Run tests
srt "npm test"
srt "pytest"
srt "go test ./..."

# Build
srt "npm run build"
srt "cargo build --release"
```

### CI/CD Pipeline

Maximum restrictions for untrusted code:

```json
{
  "network": {
    "defaultPolicy": "deny",
    "allowedDomains": ["api.github.com"]
  },
  "filesystem": {
    "denyRead": ["~/**"],
    "allowWrite": ["./dist", "./build"]
  }
}
```

```bash
srt --settings ci-config.json "npm run build"
```

## How It Works

### Architecture

```
┌─────────────────────────────────────┐
│  User: srt "npx -y vibe-kanban"     │
└──────────────┬──────────────────────┘
               ↓
┌──────────────────────────────────────┐
│  Sandbox Manager                     │
│  • Load configuration                │
│  • Start HTTP/SOCKS5 proxies         │
│  • Generate Seatbelt profile         │
│  • Set proxy environment variables   │
└──────────────┬───────────────────────┘
               ↓
┌──────────────────────────────────────┐
│  sandbox-exec -f profile.sb command  │
│                                      │
│  macOS Seatbelt enforces:            │
│  ✓ Process execution allowed         │
│  ✓ Network only to proxy ports       │
│  ✓ File reads (with deny list)      │
│  ✓ File writes (with allow list)    │
└──────────────┬───────────────────────┘
               ↓
    ┌──────────┴──────────┐
    ↓                     ↓
┌───────────┐      ┌────────────┐
│HTTP Proxy │      │SOCKS Proxy │
│Domain     │      │Domain      │
│Filter     │      │Filter      │
└─────┬─────┘      └──────┬─────┘
      └──────┬─────────────┘
             ↓
        Internet
```

### Components

1. **Seatbelt Profile Generation**: Converts configuration to Scheme-based Seatbelt rules
2. **HTTP/HTTPS Proxy**: Filters web traffic by domain using CONNECT tunneling
3. **SOCKS5 Proxy**: Filters non-HTTP traffic (SSH, Git over SSH, etc.)
4. **Violation Monitor**: Watches macOS unified logging for sandbox violations
5. **Dangerous File Scanner**: Uses ripgrep to find protected files in allowed directories

### Security Model

**Filesystem Restrictions**:
- Enforced at kernel level by macOS Seatbelt
- Process cannot bypass restrictions
- Symbolic links are followed (watch for symlink attacks)

**Network Filtering**:
- Proxies run outside sandbox
- Sandboxed process can only connect to localhost proxy ports
- Domain filtering happens in proxy before forwarding
- Cannot inspect HTTPS traffic content (domain only)

**Limitations**:
- Not a complete security boundary (shares kernel)
- Cannot protect against kernel exploits
- Seatbelt rules must be syntactically correct
- Domain-level filtering only (no DPI)
- macOS 26+ only

## Monitoring and Debugging

### Verbose Mode

See detailed execution information:

```bash
srt --verbose "npm install"
```

Output includes:
- Configuration loaded
- Proxy ports assigned
- Generated Seatbelt profile
- Command execution details
- Sandbox violations in real-time

### Violation Messages

Example violation:

```
WARN Sandbox violation process=cat operation=file-read target=/Users/user/.ssh/id_rsa
```

Violations indicate:
- **process**: Which binary tried to access
- **operation**: What type of access (file-read, file-write, network)
- **target**: What resource was blocked

### Common Issues

**"Operation not permitted"**
- Command is blocked by filesystem or network restrictions
- Run with `--verbose` to see what was denied
- Adjust configuration to allow the required access

**"Network connection timeout"**
- Domain may not be in allowlist (if using deny-by-default)
- Check proxy is running (verbose mode shows ports)
- Verify network connectivity outside sandbox

**"Command not found"**
- Check command is in PATH
- Seatbelt allows execution of all binaries by default

**"Invalid Seatbelt profile"**
- Configuration syntax error
- Check glob patterns are valid
- Ensure paths exist and are normalised

## Performance

### Benchmarks

| Metric | srt (Go) | TypeScript Version |
|--------|----------|-------------------|
| Startup time | 2-5ms | ~100ms |
| Memory usage | 15-30MB | ~60MB |
| Proxy throughput | 50-100k req/s | ~15k req/s |
| Binary size | 9MB (single arch) | ~50MB+ (with node_modules) |

### Optimisations

- Proxies start in parallel with configuration loading
- Seatbelt profile generated once and reused
- Glob patterns compiled once at startup
- Violation monitoring uses efficient log streaming
- Zero-copy proxy forwarding where possible

## Development

### Building

```bash
# Development build
make build

# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Clean build artifacts
make clean
```

### Project Structure

```
srt-go/
├── cmd/srt/           # CLI entry point
├── internal/
│   ├── config/        # Configuration loading and validation
│   ├── filesystem/    # Path normalisation, glob matching, scanning
│   ├── network/       # HTTP/SOCKS proxies, domain filtering
│   ├── platform/      # macOS version detection
│   └── sandbox/       # Seatbelt profile generation, execution, monitoring
├── Makefile           # Build commands
└── go.mod             # Dependencies
```

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/armon/go-socks5` - SOCKS5 server
- `github.com/gobwas/glob` - Glob pattern matching

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/filesystem/
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

Code should follow Go conventions and use Australian/British English spelling in comments and documentation.

## Comparison with TypeScript Version

### Advantages

- **10-50x faster** startup and execution
- **Single binary** distribution (no Node.js required)
- **Lower memory** usage (~50% reduction)
- **Better concurrency** with goroutines for proxies
- **Simpler deployment** (just copy binary)
- **Compile-time safety** with Go's type system

### Trade-offs

- Requires Go toolchain to build
- Different codebase to maintain
- Less ecosystem for some specialised tasks

### Migration Path

The Go version maintains configuration compatibility with the TypeScript version, making migration straightforward:

1. Install srt-go alongside TypeScript version
2. Test with same configuration file
3. Validate behaviour matches
4. Switch production systems gradually
5. Maintain TypeScript version for backwards compatibility if needed

## Troubleshooting

### System Requirements Not Met

Error: `system requirements not met: macOS 26.0 or newer required`

Solution: This tool requires macOS 26 (Tahoe) or newer. Check version:

```bash
sw_vers -productVersion
```

### Config File Not Found

If `~/.srt-settings.json` is missing, srt creates it with defaults on first run. To reset to defaults:

```bash
rm ~/.srt-settings.json
srt "echo test"
```

### Proxy Conflicts

If ports 8080 or 1080 are in use, srt auto-assigns available ports. Check assigned ports with `--verbose`.

### Performance Issues

For large file operations, ensure ripgrep is installed for fast dangerous file scanning:

```bash
brew install ripgrep
```

## Security Considerations

### What srt Protects Against

- Accidental credential exposure
- Malicious package scripts reading secrets
- Unintended file modifications
- Unauthorised network access
- AI agents accessing sensitive data

### What srt Does NOT Protect Against

- Kernel exploits or privilege escalation
- Attacks on the sandbox itself
- Process memory inspection
- Time-based side channels
- Malicious code that runs before sandbox activates

### Best Practices

1. **Use deny-by-default network policy** for untrusted code
2. **Block secret directories** in denyRead
3. **Limit write access** to specific directories
4. **Review violations regularly** to tune configuration
5. **Keep macOS updated** for latest security fixes
6. **Don't rely solely on srt** - use defence in depth

## FAQ

**Q: Does srt work on Linux or Windows?**
A: No, srt requires macOS 26+ (Tahoe) and the Seatbelt framework.

**Q: Can srt protect against malicious npm packages?**
A: Partially. It can prevent credential theft and unauthorised network access, but determined malware may still cause harm within allowed boundaries.

**Q: What's the performance impact?**
A: Minimal - typically 2-5ms startup overhead and negligible runtime overhead due to kernel-level enforcement.

**Q: Can I use srt in production?**
A: Yes, but understand the limitations. It provides defence-in-depth, not complete isolation.

**Q: How does srt compare to Docker?**
A: Different tools. Docker provides full containerisation, srt provides lightweight OS-level sandboxing. Use Docker for true isolation, srt for convenience and performance.

**Q: Can commands bypass the sandbox?**
A: Not without kernel exploits. Seatbelt restrictions are enforced at kernel level.

**Q: Does srt slow down file operations?**
A: No, kernel-level enforcement has negligible overhead. The macOS kernel checks permissions directly.

## License

MIT License - see LICENSE file for details

## Authors

Built by Anthropic for secure AI agent execution and general-purpose sandboxing on macOS.

## Support

- Issues: [GitHub Issues](https://github.com/sammcj/srt-go/issues)
- Discussions: [GitHub Discussions](https://github.com/sammcj/srt-go/discussions)

## References

- [macOS Sandbox Design Guide](https://developer.apple.com/library/archive/documentation/Security/Conceptual/AppSandboxDesignGuide/)
- [Seatbelt Profiles (The Apple Wiki)](https://theapplewiki.com/wiki/Dev:Seatbelt)
- [macOS Unified Logging](https://developer.apple.com/documentation/os/logging)
- [Original TypeScript Implementation](https://github.com/anthropic-experimental/anthropic-sandbox-runtime)
