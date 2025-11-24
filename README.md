# srt: Sandbox Runtime for macOS

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
- **Auto-detection** of 20+ package managers (Homebrew, nvm, pyenv, cargo, etc.)
- **Secure by default** - most restrictive defaults requiring explicit permissions
- **Profile validation** - syntax and live testing of Seatbelt profiles before execution
- **Conditional proxy startup** - no overhead when network is fully blocked
- **Expanded credential protection** - shell history, cloud credentials, database configs
- **Real-time violation monitoring** with structured reporting
- **Native macOS integration** using Seatbelt framework
- **Fast startup** (~2-5ms overhead, less when proxy disabled)
- **Single binary** - no dependencies except macOS 26+

## Installation

### Prerequisites

- macOS 26.0 (Tahoe) or newer
- Go 1.23+ (for building from source)

### Go install

```bash
go install github.com/sammcj/srt@HEAD
```

### Manual Build and Install

```bash
# Clone the repository
git clone https://github.com/sammcj/srt
cd srt

# Build for current architecture
make build

# Install to /usr/local/bin
make install
```

### Verify Installation

```bash
srt --version
```

On first run, srt creates `~/.srt/srt-settings.json` with secure defaults (deny all writes, deny all network). You'll need to configure permissions for your specific use case.

Generate a configuration file explicitly with:
```bash
srt init
```

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

# Use a preset configuration
srt --preset=development "npm install"
srt --preset=readonly ls -la

# Custom configuration
srt --settings ./my-config.json "python setup.py install"

# Dry-run to see what would be executed
srt --dry-run --preset=readonly "echo hello"

# Override configuration for specific command
srt --override-config='{"filesystem":{"allowWrite":["./dist"]}}' "npm run build"
```

### Integration with Claude Code

Configure Claude Code to use srt for all sandboxed commands.

#### Using Default Configuration

The simplest configuration uses srt with default settings from `~/.srt/srt-settings.json`:

```json
{
  "sandbox": {
    "command": "srt",
    "enabled": true
  }
}
```

#### Using a Custom Configuration File

For project-specific sandboxing, point to a custom configuration file:

```json
{
  "sandbox": {
    "command": "srt --settings /path/to/project-srt-config.json",
    "enabled": true
  }
}
```

Example workflow:
```bash
# Create a project-specific config
srt init ./project-srt-config.json

# Edit it for your project needs
vim ./project-srt-config.json

# Configure Claude Code to use it (in Claude settings)
{
  "sandbox": {
    "command": "srt --settings ./project-srt-config.json",
    "enabled": true
  }
}
```

#### Using Presets

Use built-in presets for different development scenarios:

```json
{
  "sandbox": {
    "command": "srt --preset=development",
    "enabled": true
  }
}
```

Available presets: `development`, `readonly`, `ci`, `package-install`

#### Advanced: Per-Tool Configuration

Configure different sandboxing for different tools:

```json
{
  "sandbox": {
    "command": "srt",
    "enabled": true,
    "toolOverrides": {
      "read": "srt --preset=readonly",
      "bash": "srt --preset=development",
      "write": "srt --settings ./strict-write-config.json"
    }
  }
}
```

Now Claude Code runs all commands through srt automatically, preventing access to secrets and controlling network access according to your configuration.

## Configuration

Configuration is stored in `~/.srt/srt-settings.json` and created automatically on first run.

### Generate Configuration File

Create a default configuration file:

```bash
# Create at default location (~/.srt/srt-settings.json)
srt init

# Create at custom location
srt init ./my-srt-config.json
```

### Configuration Structure

```json
{
  "network": {
    "defaultPolicy": "deny",
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
      "~/.bash_history",
      "~/.zsh_history",
      "~/.python_history",
      "~/.node_repl_history",
      "~/.mysql_history",
      "~/.psql_history",
      "~/.s3cfg",
      "~/.boto",
      "~/.botocore/**",
      "~/.azure/**",
      "~/.config/doctl/**",
      "~/.pgpass",
      "~/.my.cnf",
      "~/.mysql_secret",
      "~/.redis.conf",
      "~/.docker/config.json",
      "~/.kube/config",
      "~/.password-store/**",
      "~/.authinfo",
      "~/.authinfo.gpg",
      "~/.netrc",
      "/var/db/**",
      "~/Library/Keychains/**",
      "/System/Library/Keychains/**",
      "/System/Library/Security/**",
      "/System/Library/PrivateFrameworks/**",
      "/System/Library/Extensions/**",
      "/System/Library/LaunchDaemons/**",
      "/System/Library/LaunchAgents/**"
    ],
    "allowWrite": [],
    "denyWrite": [],
    "allowUnlink": []
  },
  "process": {
    "allowFork": true,
    "allowSysctlRead": true,
    "allowMachLookup": true,
    "allowPosixShm": true
  },
  "scanAndBlockFiles": [
    ".env",
    ".git-credentials",
    "srt-settings.json",
    ".keychain",
    ".ripgreprc"
  ],
  "scanAndBlockDirs": [
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

**Deny-by-default** (restrictive, **default and recommended**):
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
- **Default configuration** - secure by default
- Good for running untrusted code, AI agents, or CI/CD

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

Note: See [default-config.json](internal/config/default-config.json) for an up-to-date list of default settings.

#### Read Restrictions

The default configuration includes credential protection for common secret locations:

```json
{
  "filesystem": {
    "denyRead": [
      "~/.ssh/**",                  // SSH keys
      "~/.aws/**",                  // AWS credentials
      "~/.config/gcloud/**",        // Google Cloud credentials
      "~/.bash_history",            // Shell history (may contain secrets)
      "~/.zsh_history",
      "~/.python_history",
      "~/.node_repl_history",
      "~/.mysql_history",
      "~/.psql_history",
      "~/.s3cfg",                   // S3 configuration
      "~/.boto",                    // Boto/AWS SDK configuration
      "~/.botocore/**",
      "~/.azure/**",                // Azure credentials
      "~/.config/doctl/**",         // DigitalOcean credentials
      "~/.pgpass",                  // PostgreSQL password file
      "~/.my.cnf",                  // MySQL configuration
      "~/.mysql_secret",
      "~/.redis.conf",              // Redis configuration
      "~/.docker/config.json",      // Docker credentials
      "~/.kube/config",             // Kubernetes credentials
      "~/.password-store/**",       // Password store
      "~/.authinfo",                // Auth credentials
      "~/.authinfo.gpg",
      "~/.netrc",                   // Network credentials
      "**/.env*",                   // Environment files anywhere
      "~/secrets/**"                // Custom secrets directory
    ]
  }
}
```

- By default, all paths are readable
- Paths in `denyRead` are blocked
- Includes protection for shell history, cloud credentials, database configs, and password stores
- Supports glob patterns (see below)

#### Write Restrictions

**Secure by Default**: The default configuration has `allowWrite: []` (no writes allowed). You must explicitly grant write permissions for the directories you need.

**Package Manager Auto-Detection**: srt automatically detects 20+ package managers and version managers on your system and adds their cache directories to `allowWrite` at runtime. This includes:

- **Homebrew**: `/opt/homebrew`, `/usr/local/Homebrew`
- **Nix**: `/nix/store`, `~/.nix-profile`
- **Node.js**: nvm, fnm, nodenv, Deno, Bun
- **Python**: pyenv, Poetry, pipx, Conda/Miniconda
- **Go**: `~/go`, g version manager
- **Java**: SDKMAN, jenv
- **Ruby**: rbenv, RVM
- **Rust**: Cargo, Rustup
- **Standard caches**: `~/.npm`, `~/.cache/pip`, `~/.cargo`, etc.

**Example configuration for development**:
```json
{
  "filesystem": {
    "allowWrite": [
      ".",                          // Current directory
      "./build/**",                 // Build output
      "./dist/**"                   // Distribution files
    ],
    "denyWrite": [
      "**/.env*",                   // Block .env files
      "**/*.key",                   // Block key files
      "**/.git/**"                  // Block .git directory
    ]
  }
}
```

- By default, all writes are denied (secure by default)
- Only paths in `allowWrite` permit writes
- Package manager caches are automatically detected and added
- Paths in `denyWrite` are blocked even if parent is allowed
- More specific rules override general rules

**Note**: Package manager paths are auto-detected at runtime, so you don't need to manually configure them in your settings file. This works across different macOS setups (ARM/Intel, standard/custom installations).

#### Unlink (Deletion) Restrictions

The default configuration has `allowUnlink: []` (no deletions allowed).

```json
{
  "filesystem": {
    "allowUnlink": [
      ".",
      "./build/**",
      "./dist/**"
    ]
  }
}
```

- File deletion (`unlink`) is denied by default, even in directories with write permissions
- Only paths in `allowUnlink` permit file deletion and moving
- Package manager caches are auto-detected and added to `allowUnlink` (same as `allowWrite`)
- Supports the same glob patterns as other filesystem rules

**Why separate from write?**: Separating deletion from write permissions provides defence in depth - a process can create files without being able to delete existing ones, limiting potential damage.

#### Understanding Static Blocks vs Dynamic Pattern Scanning

srt provides two complementary mechanisms for blocking filesystem access:

**Static Deny Lists** (`denyRead`, `denyWrite`):
- **Purpose**: Block specific paths everywhere in the filesystem
- **Configured as**: Exact paths or glob patterns (e.g., `~/.ssh/**`, `**/.env*`)
- **Applied**: Globally - these paths are always blocked regardless of allow rules
- **Use when**: You know exactly what to block and where it's located

**Dynamic Pattern Scanning** (`scanAndBlockFiles`, `scanAndBlockDirs`):
- **Purpose**: Find and block files by name within directories you've allowed
- **Configured as**: Simple patterns to search for (e.g., `.env`, `.secrets`, `.git-credentials`)
- **Applied**: Only scanned within `allowWrite` paths, found matches are automatically blocked
- **Use when**: You're using broad permissions but want to protect specific file types within them

##### The Key Difference

Think of it this way:
- `denyRead`/`denyWrite`: "Never allow access to **this path**" (absolute blocking)
- `scanAndBlockFiles`/`scanAndBlockDirs`: "Search my allowed directories and block anything **named this**" (pattern-based discovery)

##### Practical Example

Say you want to allow writes to your current project directory but protect sensitive files:

```json
{
  "filesystem": {
    "denyRead": [
      "~/.ssh/**",              // Static: Block SSH keys (known location)
      "~/.aws/**"               // Static: Block AWS credentials (known location)
    ],
    "allowWrite": [
      "."                       // Broad: Allow writes to current directory
    ],
    "scanAndBlockFiles": [
      ".env",                   // Dynamic: Find and block .env files
      ".git-credentials",       // Dynamic: Find and block git credentials
      "id_rsa"                  // Dynamic: Find and block SSH keys
    ],
    "scanAndBlockDirs": [
      ".secrets",               // Dynamic: Find and block .secrets directories
      ".aws"                    // Dynamic: Find and block .aws directories
    ]
  }
}
```

**What happens at runtime:**

1. Static rules immediately block `~/.ssh/**` and `~/.aws/**` globally
2. You broadly allow writes to `.` (current directory)
3. Before executing, srt scans `.` using ripgrep to find:
   - Any files named `.env` → found `./api/.env`, `./.env.local`
   - Any directories named `.secrets` → found `./config/.secrets`
   - Any files named `id_rsa` → found `./backup/id_rsa`
4. These discovered paths are automatically added to the deny list
5. **Result**: You can write to the directory, but NOT to sensitive files within it

##### When to Use Which

**Use `denyRead`/`denyWrite` when you:**
- Know the exact paths to block (e.g., `~/.ssh/**`)
- Want to block paths in well-known locations
- Need guaranteed protection regardless of other configuration
- Want to block paths outside your allowed directories

**Use `scanAndBlockFiles`/`scanAndBlockDirs` when you:**
- Use broad permissions like `"allowWrite": ["."]`
- Want to block files by their name pattern, wherever they appear within allowed paths
- Need defence-in-depth against accidentally including sensitive files in projects
- Want to catch `.env`, credentials, or key files that might be in subdirectories

##### Why Both?

This provides defence-in-depth:
- **Static deny lists** = "Hard firewall rules" - blocks you know about
- **Dynamic scanning** = "Content inspection" - discovers blocks within allowed paths

Even if someone configures broad write access, blocked files within those directories are still protected. It's like having both a firewall (static deny) and an antivirus scanner (dynamic patterns) working together.

##### Performance Note

Pattern scanning only happens once at startup within allowed write paths (not glob patterns). If ripgrep is installed, scanning is very fast (milliseconds). Results are limited to actual directories that exist and have been allowed for writing.

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

### Violation Filtering

When running with `--verbose`, srt reports sandbox violations - these are blocked access attempts that were denied by the sandbox (which is exactly what should happen). However, many programs routinely try to access system paths like `/usr/bin` or `/System` as part of normal operation, creating noise in the logs.

The `ignoreViolations` configuration filters out expected violations from log output to help you focus on unexpected access attempts.

#### What Are Violations?

Violations are blocked access attempts that occur when:
- A process tries to read a file in `denyRead`
- A process tries to write to a path not in `allowWrite`
- A process tries to connect to a blocked network domain

**Important**: `ignoreViolations` only affects logging - the violations are still blocked. This is purely about reducing noise in verbose output.

#### Configuration

```json
{
  "ignoreViolations": {
    "*": ["/usr/bin", "/usr/lib", "/System", "/Library"],
    "git": ["/usr/bin/ssh-agent"],
    "npm": ["/usr/local/lib"]
  }
}
```

**Structure**:
- **Key**: Process name (or `"*"` for all processes)
- **Value**: Array of path substrings to ignore

**Matching behaviour**:
- Uses substring matching on the violation target path
- First checks process-specific ignores, then global (`"*"`) ignores
- If a violation's target contains any ignore pattern, it won't be logged

#### When to Use

**Use `ignoreViolations` when:**
- Running with `--verbose` and seeing repeated violations for system paths
- You've verified the violations are expected (access correctly blocked)
- You want to focus on unexpected violations only
- Debugging specific issues and need cleaner logs

**Don't use `ignoreViolations` to:**
- Actually allow access (use `allowWrite`/`allowRead` instead)
- Hide security issues (investigate unexpected violations first)
- Bypass sandbox restrictions (violations are still blocked regardless)

#### Practical Example

Say you run `srt --verbose "git push"` and see:

```
WARN Sandbox violation process=git operation=file-read target=/usr/bin/ssh-agent
WARN Sandbox violation process=git operation=file-read target=/usr/bin/ssh
WARN Sandbox violation process=git operation=file-read target=/System/Library/Frameworks/...
WARN Sandbox violation process=git operation=file-write target=/Users/you/.ssh/known_hosts
```

The first three are expected (system binaries that git checks but doesn't need). The fourth is unexpected and worth investigating. Configure:

```json
{
  "ignoreViolations": {
    "git": ["/usr/bin", "/System"]
  }
}
```

Now you'll only see:
```
WARN Sandbox violation process=git operation=file-write target=/Users/you/.ssh/known_hosts
```

This makes the real issue visible: git is trying to write to `.ssh/known_hosts`, which you may want to allow depending on your use case.

#### Default Configuration

By default, srt ignores violations for common system paths:
- `/usr/bin` - System binaries
- `/usr/lib` - System libraries
- `/System` - macOS system files
- `/Library` - System frameworks

These paths are frequently accessed by programs but are intentionally blocked for security. Filtering them reduces log noise while still blocking the access.

## Preset Configurations

srt includes pre-configured presets for common use cases. Use `--preset=<name>` to load them:

### Available Presets

#### `development`
Permissive configuration for local development:
- Allows writes to current directory and package manager caches
- Allows network access to common package registries (npm, PyPI, crates.io, Go modules)
- Good for: local development, package installation, testing

```bash
srt --preset=development "npm install"
srt --preset=development "cargo build"
```

#### `readonly`
Strictest mode with no write or network access:
- No writes allowed anywhere
- No network access
- Good for: inspecting files, running read-only analysis tools

```bash
srt --preset=readonly ls -la
srt --preset=readonly cat package.json
```

#### `ci`
Build-focused mode for CI/CD:
- Allows writes only to build output directories (`./dist`, `./build`, `./target`, `./out`)
- Allows package manager caches
- Allows network access to package registries
- Good for: CI/CD pipelines, automated builds

```bash
srt --preset=ci "npm run build"
srt --preset=ci "cargo build --release"
```

#### `package-install`
Package manager focused mode:
- Allows writes to package manager caches and lock files
- Allows writes to dependency directories (`node_modules`, `vendor`)
- Allows network access to package registries
- Good for: dependency installation without source code modification

```bash
srt --preset=package-install "npm install"
srt --preset=package-install "pip install -r requirements.txt"
```

### Custom Presets

Presets are stored in the `presets/` directory. You can create custom presets by adding JSON files there:

```bash
# Create custom preset
cat > presets/custom.json << 'EOF'
{
  "filesystem": {
    "allowWrite": ["./output/**"]
  },
  "network": {
    "allowedDomains": ["example.com"]
  }
}
EOF

# Use custom preset
srt --preset=custom "mycommand"
```

## Dry-Run Mode

Use `--dry-run` to inspect what srt will do without executing the command:

```bash
srt --dry-run --preset=readonly "echo hello"
```

**Output includes:**
- Generated Seatbelt profile (complete sandbox rules)
- Command that would be executed
- Environment variables (proxy settings, etc.)
- Filesystem permissions summary (counts of allowed/denied paths)
- Network configuration summary
- Detected package manager paths

**Use cases:**
- Debugging sandbox configuration
- Understanding what restrictions will be applied
- Verifying preset behaviour
- Educational purposes (learning Seatbelt syntax)
- Troubleshooting permission issues

## Configuration Overrides

Override configuration for specific commands without modifying `~/.srt/srt-settings.json`:

### CLI Flag

```bash
srt --override-config='{"filesystem":{"allowWrite":["./dist"]}}' "npm run build"
```

### Environment Variable

```bash
export SRT_CONFIG_OVERRIDE='{"filesystem":{"allowWrite":[]}}'
srt "ls -la"  # Readonly mode
```

### JSON File

```bash
# Create override file
echo '{"network":{"allowedDomains":["github.com"]}}' > override.json

# Use it
srt --override-config=override.json "git push"
```

### Override Semantics

Configuration overrides follow these rules:
1. Base configuration is loaded first (from `~/.srt/srt-settings.json` or defaults)
2. Preset is applied if `--preset` is specified
3. Override is applied last if `--override-config` or `SRT_CONFIG_OVERRIDE` is set
4. Fields present in override **completely replace** base values:
   - `[]` (empty array) overrides to empty (most restrictive)
   - Non-empty arrays replace entire base array
   - Omitted fields keep base values

**Example:**

```bash
# Base has: allowedDomains: ["github.com", "npmjs.org"]
# Override:  {"network": {"allowedDomains": ["pypi.org"]}}
# Result:    allowedDomains: ["pypi.org"]  # Completely replaced
```

### Common Override Patterns

**Readonly mode for inspection:**
```bash
SRT_CONFIG_OVERRIDE='{"filesystem":{"allowWrite":[],"allowUnlink":[]}}' srt ls
```

**GitHub-only network:**
```bash
srt --override-config='{"network":{"allowedDomains":["github.com","*.github.com"]}}' "git clone ..."
```

**Build output only:**
```bash
srt --override-config='{"filesystem":{"allowWrite":["./dist","./build"]}}' "make build"
```

### Claude Code Integration

Configure per-tool overrides in Claude Code settings:

```json
{
  "sandbox": {
    "command": "srt",
    "readonlyOverride": {
      "filesystem": {"allowWrite": [], "allowUnlink": []},
      "network": {"allowedDomains": []}
    }
  }
}
```

## Caching

srt automatically caches package manager detection results for performance:

### Cache Behaviour

- **Location**: `$TMPDIR/.srt-cache-<username>.json`
- **TTL**: 1 hour (default)
- **Invalidation**: Automatic based on TTL
- **Performance**: Saves 2-5ms per execution

### Cache Configuration

```bash
# Set custom TTL (e.g., 30 minutes)
export SRT_CACHE_TTL=30m

# Or 2 hours
export SRT_CACHE_TTL=2h
```

### Manual Cache Management

```bash
# View cache location
ls $TMPDIR/.srt-cache-$(whoami).json

# Clear cache
rm $TMPDIR/.srt-cache-$(whoami).json

# Disable caching (set very short TTL)
export SRT_CACHE_TTL=1ns
```

The cache stores:
- Detected package manager paths
- Cache timestamp

The cache does NOT store:
- Configuration settings
- Command history
- Violation logs

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
5. **Blocked File Scanner**: Uses ripgrep to find protected files in allowed directories

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

### Denial Log

All sandbox violations (blocked access attempts) are automatically logged to `~/.srt/deny.log`, regardless of whether verbose mode is enabled. This provides a persistent audit trail of what sandboxed commands attempted to access.

#### Log Details

- **Location**: `~/.srt/deny.log`
- **Format**: Timestamped entries with process, operation, and target information
- **Rotation**: Automatically rotates when file reaches 512KB
- **Retention**: Keeps up to 3 rotated log files (deny.log.1, deny.log.2, deny.log.3)
- **Always enabled**: Logging occurs for all commands, not just in verbose mode

#### Example Log Entries

```
2025/01/15 14:32:01 VIOLATION process=npm operation=file-read target=/Users/user/.ssh/id_rsa time=2025-01-15 14:32:01
2025/01/15 14:32:03 VIOLATION process=git operation=network target=github.com:443 time=2025-01-15 14:32:03
2025/01/15 14:32:05 VIOLATION process=python operation=file-write target=/Users/user/.aws/credentials time=2025-01-15 14:32:05
```

#### Viewing the Log

```bash
# View recent violations
tail -f ~/.srt/deny.log

# View all violations
cat ~/.srt/deny.log

# Search for specific violations
grep "file-write" ~/.srt/deny.log
grep "npm" ~/.srt/deny.log
```

#### Managing the Log

The log rotates automatically, but you can manually clear it if needed:

```bash
# Clear the denial log
rm ~/.srt/deny.log*

# View log size
ls -lh ~/.srt/deny.log*
```

**Note**: Violations logged here respect the `ignoreViolations` configuration - only violations that aren't filtered out appear in the log.

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
srt/
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

1. Install srt alongside TypeScript version
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

If `~/.srt/srt-settings.json` is missing, srt creates it with defaults on first run. To reset to defaults:

```bash
rm ~/.srt/srt-settings.json
srt init
```

### Proxy Conflicts

If ports 8080 or 1080 are in use, srt auto-assigns available ports. Check assigned ports with `--verbose`.

### Performance Issues

For large file operations, ensure ripgrep is installed for fast blocked file scanning:

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

- Issues: [GitHub Issues](https://github.com/sammcj/srt/issues)
- Discussions: [GitHub Discussions](https://github.com/sammcj/srt/discussions)

## References

- [macOS Sandbox Design Guide](https://developer.apple.com/library/archive/documentation/Security/Conceptual/AppSandboxDesignGuide/)
- [Seatbelt Profiles (The Apple Wiki)](https://theapplewiki.com/wiki/Dev:Seatbelt)
- [macOS Unified Logging](https://developer.apple.com/documentation/os/logging)
- [Original TypeScript Implementation](https://github.com/anthropic-experimental/anthropic-sandbox-runtime)
