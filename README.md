# 🐙 Octo

**Orchestrate your Docker containers like an octopus.**

Octo is a powerful Docker container management CLI tool that helps you monitor, analyze, and clean up Docker resources with an intuitive interface.

## Features

- **📊 Status Dashboard** - Real-time Docker system monitoring
- **🔍 Resource Analyzer** - Interactive exploration of containers, images, volumes
- **🧹 Smart Cleanup** - Safely remove unused resources with dry-run mode
- **🗑️ Deep Prune** - Comprehensive cleanup of all unused Docker resources
- **🩺 Health Diagnostics** - Check Docker daemon health and configuration
- **🎨 Beautiful TUI** - Terminal user interface with keyboard navigation

## Quick Start

### Installation

**Via Install Script (Recommended)**
```bash
curl -fsSL https://raw.githubusercontent.com/bsisduck/octo/main/install.sh | bash
```

**Via Go**
```bash
go install github.com/bsisduck/octo@latest
```

**From Source**
```bash
git clone https://github.com/bsisduck/octo.git
cd octo
make install
```

### Basic Usage

```bash
# Launch interactive menu
octo

# Show Docker status
octo status

# Watch status in real-time
octo status -w

# Analyze Docker resources
octo analyze

# Smart cleanup (with confirmation)
octo cleanup

# Preview cleanup without making changes
octo cleanup --dry-run

# Deep prune all unused resources
octo prune

# Run diagnostics
octo diagnose

# Show version info
octo version
```

## Commands

### `octo` (Interactive Menu)

Launch the interactive TUI menu for navigating all features:

```
   ___       _
  / _ \  ___| |_ ___
 | | | |/ __| __/ _ \
 | |_| | (__| || (_) |
  \___/ \___|\__\___/

  Quick Stats
  Containers:  5 (3 running)
  Images:      12
  Volumes:     8
  Disk Used:   2.5 GB
  Reclaimable: 850 MB

  Commands
▸ 1. Status      Monitor system health
  2. Analyze     Explore resource usage
  3. Cleanup     Smart cleanup with safety
  4. Prune       Deep cleanup all unused
  5. Diagnose    Check Docker health
```

### `octo status`

Display Docker system status and resource usage:

```bash
octo status          # One-time status display
octo status -w       # Continuous monitoring mode
```

### `octo analyze`

Interactive resource analyzer with drill-down navigation:

```bash
octo analyze                    # Full overview
octo analyze -t images          # Focus on images
octo analyze -t containers      # Focus on containers
octo analyze -t volumes         # Focus on volumes
octo analyze --dangling         # Show only unused resources
```

**Navigation:**
- `↑/↓` or `j/k` - Move selection
- `Enter` or `l` - Drill down into category
- `h` or `←` - Go back
- `d` - Delete selected resource
- `r` - Refresh
- `q` - Quit

### `octo cleanup`

Smart cleanup with safety checks and confirmation prompts:

```bash
octo cleanup                    # Clean all unused resources
octo cleanup --dry-run          # Preview without changes
octo cleanup --containers       # Remove stopped containers only
octo cleanup --images           # Remove dangling images only
octo cleanup --volumes          # Remove unused volumes only
octo cleanup --networks         # Remove unused networks only
octo cleanup --build-cache      # Clear build cache only
octo cleanup --all              # Remove ALL unused (not just dangling)
octo cleanup --force            # Skip confirmation prompts
```

### `octo prune`

Deep cleanup equivalent to `docker system prune -a`:

```bash
octo prune                      # Prune with confirmation
octo prune --dry-run            # Preview what would be removed
octo prune --volumes            # Also remove anonymous volumes
octo prune --all                # Remove all unused images
octo prune --force              # Skip confirmation
```

### `octo diagnose`

Health check and diagnostics:

```bash
octo diagnose                   # Run all diagnostic checks
octo diagnose --verbose         # Show detailed results
```

**Checks performed:**
- Docker connection
- Docker version and API
- Storage driver
- Disk usage
- Container/image/volume counts
- API responsiveness
- Memory configuration

## Global Options

```bash
--debug       Enable debug output
--dry-run     Preview changes without executing
--no-color    Disable colored output
```

## Configuration

Octo uses sensible defaults and auto-detects Docker socket location:

| Platform | Default Socket Locations |
|----------|-------------------------|
| macOS    | `~/.docker/run/docker.sock`, `/var/run/docker.sock` |
| Linux    | `/var/run/docker.sock`, `$XDG_RUNTIME_DIR/docker.sock` |
| Windows  | `\\.\pipe\docker_engine` |

Override with `DOCKER_HOST` environment variable:
```bash
export DOCKER_HOST=unix:///var/run/docker.sock
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/k` | Move up |
| `↓/j` | Move down |
| `Enter` | Select/Drill down |
| `←/h` | Go back |
| `d` | Delete selected |
| `r` | Refresh |
| `q/Esc` | Quit |

## Tips

### Preview Before Cleaning
Always use `--dry-run` first to see what would be removed:
```bash
octo cleanup --dry-run
```

### Quick Monitoring
Use watch mode for continuous monitoring:
```bash
octo status -w
```

### Find Large Images
Use the analyzer to sort images by size:
```bash
octo analyze -t images
```

### Safe Cleanup Workflow
1. Run `octo diagnose` to check Docker health
2. Run `octo analyze` to understand resource usage
3. Run `octo cleanup --dry-run` to preview
4. Run `octo cleanup` to execute

## Building from Source

### Requirements
- Go 1.21 or later
- Make

### Build
```bash
git clone https://github.com/bsisduck/octo.git
cd octo
make build
```

### Install
```bash
make install    # Installs to /usr/local/bin
```

### Release Builds
```bash
make release    # Builds for all platforms
```

## Project Structure

```
octo/
├── main.go              # Entry point
├── cmd/                 # Command implementations
│   ├── root.go         # Root command and CLI setup
│   ├── docker_client.go # Docker API wrapper
│   ├── menu.go         # Interactive menu TUI
│   ├── status.go       # Status command
│   ├── analyze.go      # Analyze command
│   ├── cleanup.go      # Cleanup command
│   ├── prune.go        # Prune command
│   ├── diagnose.go     # Diagnose command
│   └── version.go      # Version command
├── bin/                 # Built binaries
├── tests/              # Test files
├── Makefile            # Build automation
├── install.sh          # Installation script
└── README.md           # This file
```

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Inspired by [Mole](https://github.com/tw93/mole) - macOS optimization toolkit
- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- Uses [Docker SDK for Go](https://github.com/docker/docker) - Docker API client
