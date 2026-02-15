# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build          # Build bin/octo (CGO_ENABLED=0, static binary)
make test           # Run all tests: go test -v ./...
make lint           # Run golangci-lint (errcheck, govet, staticcheck, goimports, etc.)
make fmt            # Format with gofmt
make fmt-check      # Verify formatting without changes
make test-coverage  # Generate coverage.html report
make run            # Build and run the binary
```

Run a single test:
```bash
go test -v -run TestFunctionName ./internal/docker/
```

Integration tests require a built binary:
```bash
make build && go test -v ./tests/
```

Version info is injected via ldflags (`cmd.Version`, `cmd.BuildTime`, `cmd.GitCommit`).

## Architecture

**Octo** is a Docker management CLI with both direct commands and an interactive TUI menu. Module path: `github.com/bsisduck/octo`.

### Package Layout

- `main.go` - Entry point with global panic recovery that restores terminal state (cursor, alt-screen, colors)
- `cmd/` - Cobra commands. `root.go` defines global flags (`--debug`, `--dry-run`, `--no-color`, `--output-format`) and dispatches the TUI menu when run without subcommands. Menu dispatches selected actions by calling `cmd.RunE()` directly (not re-entering `Execute()`)
- `internal/docker/` - Docker client abstraction (the core of the project)
- `internal/tui/` - Bubble Tea models for each view (`menu/`, `status/`, `analyze/`)
- `internal/tui/common/` - Shared TUI components (keybindings, messages)
- `internal/ui/styles/` - Centralized Lipgloss theme. All color definitions live here. `DisableColors()` zeroes styles for `--no-color`/`NO_COLOR`
- `internal/ui/format/` - Output formatters (text/JSON/YAML) selected by `--output-format` flag

### Two-Layer Docker Interface

The Docker abstraction uses two interfaces in `internal/docker/interface.go`:

1. **`DockerAPI`** - Thin wrapper matching the Docker SDK's `*client.Client` method signatures. Used for testing data transformation logic (raw API types to domain types)
2. **`DockerService`** - High-level domain operations (~30 methods). All commands and TUI models depend on this interface

`Client` in `client.go` implements `DockerService` by wrapping a `DockerAPI`. Every method creates a timeout-aware context using constants from `timeout.go`.

### Safety System

Destructive operations have three layers:
1. **TOCTOU protection** - Re-checks resource state before deletion (e.g., `RemoveContainer` re-fetches container list to verify it's still stopped)
2. **5-tier confirmation** - `SafetyTier` in `types.go`: Informational → LowRisk → Moderate → HighRisk → BulkDestructive
3. **DryRun variants** - Every destructive method has a `*DryRun` counterpart returning `ConfirmationInfo` with tier, warnings, and undo instructions

### Testing Pattern

Hand-rolled mocks in `internal/docker/mock.go`. `MockDockerService` has function fields for each `DockerService` method. Compile-time interface check: `var _ DockerService = (*MockDockerService)(nil)`.

Usage pattern:
```go
mock := &docker.MockDockerService{
    ListContainersFn: func(ctx context.Context, all bool) ([]docker.ContainerInfo, error) {
        return []docker.ContainerInfo{{ID: "abc", Name: "test"}}, nil
    },
}
```

When adding new methods to `DockerService`:
1. Add to `DockerService` interface in `interface.go`
2. If it wraps a new SDK call, add to `DockerAPI` interface too
3. Implement in `client.go`
4. Add function field + method to `MockDockerService` in `mock.go`
5. Add timeout constant in `timeout.go` if needed

### Linter Configuration

`.golangci.yml` requires `goimports` with local prefix `github.com/bsisduck/octo` (stdlib, then external, then local imports). Tests are exempt from `errcheck` and `unparam`.

### Environment Variables

- `OCTO_DEBUG=1` - Enable debug output and stack traces on panic
- `OCTO_DRY_RUN=1` - Force dry-run mode
- `NO_COLOR` - Disable colored output (standard convention)
- `DOCKER_HOST` - Override Docker socket path (falls back to platform-specific auto-detection in `socket.go`)
