# Octo

## What This Is

Octo is a Docker container management CLI tool built in Go that provides an interactive, mouse-enabled TUI for monitoring, analyzing, and safely managing Docker resources. It replaces scattered `docker` CLI commands with a unified interface that makes dangerous operations safe and routine operations fast.

## Core Value

Users can confidently manage their Docker environment without fear of accidentally destroying important resources — every destructive action shows what will happen, asks for confirmation, and explains reversibility.

## Requirements

### Validated

- ✓ Cross-platform Docker socket auto-detection (macOS, Linux, Windows) — existing
- ✓ Interactive TUI menu with Bubble Tea — existing
- ✓ Container, image, volume, network listing — existing
- ✓ Disk usage analysis and breakdown — existing
- ✓ Smart cleanup with dry-run mode — existing
- ✓ Deep prune with before/after comparison — existing
- ✓ 10-point health diagnostics — existing
- ✓ Watch mode for live status monitoring — existing
- ✓ Cobra CLI with subcommands — existing
- ✓ Cross-platform release builds — existing

### Active

- [ ] Fix critical bugs (menu dispatch, panics on short IDs, division by zero, empty name crash)
- [ ] Docker client interface for testability and mocking
- [ ] Per-operation context with timeouts (replace single background context)
- [ ] Surface swallowed errors as user-visible warnings
- [ ] Network deletion support in analyze TUI
- [ ] Align Go version across go.mod, CI, and linter config
- [ ] Mouse click support for TUI elements (select items by clicking)
- [ ] Multiple input methods: click, number keys, arrow keys, enter
- [ ] Full terminal text selection with auto-copy to clipboard
- [ ] Container logs viewer (view and tail logs from TUI)
- [ ] Docker Compose awareness (show compose projects, start/stop stacks)
- [ ] Container exec/shell (open shell inside running container)
- [ ] Safety confirmation for all destructive operations (show what will happen, explain reversibility)
- [ ] `--json` output flag for all commands (machine-readable for scripting)
- [ ] `--no-color` flag actually works (currently declared but never checked)
- [ ] Centralized style system (eliminate duplicate style definitions)
- [ ] Proper package structure (split cmd into internal/docker, internal/tui, cmd)
- [ ] 90%+ test coverage with mocked Docker client
- [ ] Unit tests for all business logic
- [ ] Integration tests that don't require pre-built binary
- [ ] Shell completion support (bash, zsh, fish)

### Out of Scope

- Real-time chat or collaboration features — this is a local CLI tool
- Docker Swarm management — focus on single-host Docker
- Kubernetes support — different tool, different scope
- Web UI or API server — terminal-only
- Plugin/extension system — premature for v1
- Container image building — use docker build directly
- Registry management — use docker push/pull directly

## Context

Octo is a brownfield project with ~2,500 LOC across 11 Go files, all in a single `cmd` package. A comprehensive code analysis identified 30 issues (4 critical, 8 high, 12 medium, 6 low). The codebase has solid foundations (Cobra, Bubble Tea, Docker SDK) but needs architectural cleanup before adding features.

Key technical debt:
- No Docker client interface — impossible to unit test without running daemon
- Single `context.Background()` reused for all operations — no timeouts, no cancellation
- Errors silently swallowed in TUI fetch operations
- All code in one package (`cmd`) — no separation of concerns
- Duplicate style definitions scattered across 6+ files
- Tests only cover pure functions (~5% real behavioral coverage)

Prior analysis saved at: `.planning/codebase/ANALYSIS.md`

## Constraints

- **Tech stack**: Go, Cobra CLI, Bubble Tea TUI, Lipgloss styling, Docker SDK v27.5.1 — established, don't replace
- **Compatibility**: Must work on macOS, Linux, Windows
- **Dependencies**: Minimize new dependencies — prefer standard library where possible
- **Binary size**: Keep static binary (CGO_ENABLED=0)
- **Go version**: Align to single version across all configs
- **Test coverage**: Target 90%+ with mocked Docker client

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Keep single binary CLI (no daemon/server) | Simplicity, no background processes | — Pending |
| Mouse click-to-select (not full mouse interaction) | Balance between usability and TUI complexity | — Pending |
| Confirm + undo info for destructive ops (not multi-step) | Good safety without being annoying | — Pending |
| `--json` for all commands | Enable scripting and tool integration | — Pending |
| 90%+ test coverage target | Ensure reliability for a tool that deletes Docker resources | — Pending |
| Split into internal packages | Testability, separation of concerns, maintainability | — Pending |

---
*Last updated: 2026-02-06 after initialization*
