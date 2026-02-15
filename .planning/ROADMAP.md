# Roadmap: Octo

## Overview

Octo is a brownfield Go CLI/TUI (~2,500 LOC) for Docker container management that currently has 4 critical crash bugs, zero test infrastructure, and all code in a single package. This roadmap takes the project from "crashes on edge-case Docker data and has a broken menu" to a stable, well-tested tool with container lifecycle management, a 5-tier safety system, log viewing, filtering, mouse support, clipboard copy, container exec/shell, and Docker Compose awareness. The approach is refactor-first (Phases 1-2), then features layered on a solid architecture (Phases 3-6).

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Critical Bug Fixes and Terminal Safety** - Fix 4 crash bugs, add panic recovery, restore terminal on signal
- [x] **Phase 2: Architecture Foundation** - Extract Docker interface, add context/timeouts, centralize styles, split packages, test infrastructure
- [ ] **Phase 3: Core Container Operations and Safety System** - Container start/stop/restart, 5-tier safety confirmations, TOCTOU protection, dry-run everywhere
- [ ] **Phase 4: Viewing Features** - Logs viewer, metrics, filtering/search, JSON output, loading indicators, fix volume sizes
- [x] **Phase 5: Interactive Enhancements** - Mouse click-to-select, clipboard copy, multiple input methods, shell completion
- [x] **Phase 6: Advanced Features** - Container exec/shell with PTY handling, Docker Compose awareness

## Phase Details

### Phase 1: Critical Bug Fixes and Terminal Safety
**Goal**: Users can run Octo without encountering crashes or terminal corruption, and the interactive menu actually dispatches the selected command
**Depends on**: Nothing (first phase)
**Requirements**: R01, R02, R03, R04, R05
**Success Criteria** (what must be TRUE):
  1. User selects an action from the interactive menu and that action executes (not silently exits)
  2. Running `octo diagnose` on a fresh Docker install with zero resources completes without panic or garbled output
  3. Listing images with short or malformed IDs does not crash the application
  4. Listing stopped containers with empty names does not crash the application
  5. Any unrecovered panic anywhere in the application restores the terminal to a clean state (cursor visible, echo on, alt-screen exited) before printing the error
**Plans**: 1 plan

Plans:
- [x] 01-01-PLAN.md -- Fix crash bugs (C1-C4) and add global panic recovery with terminal restoration

**Key Risks**:
- Pitfall CP5: Panic during Bubble Tea lifecycle leaves terminal in raw/alt-screen mode. Must install `recover()` wrapper and handle SIGTERM/SIGINT with `p.ReleaseTerminal()`.
- Pitfall CP4: Four confirmed slice-out-of-bounds locations. Bounds-check all string slicing operations.

---

### Phase 2: Architecture Foundation
**Goal**: The codebase is testable, well-structured, and every Docker operation has proper timeout and error handling -- enabling all subsequent feature work
**Depends on**: Phase 1
**Requirements**: R06, R07, R08, R09, R10, R11, R25
**Success Criteria** (what must be TRUE):
  1. All Docker operations go through a `DockerService` interface, and a complete mock implementation exists that enables testing without a running Docker daemon
  2. Every Docker API call uses a per-operation `context.Context` with a timeout (no stored `context.Background()` reused across operations)
  3. When a Docker API call fails during a TUI fetch, the user sees a warning message (not empty/incomplete data with no explanation)
  4. Code is organized into `internal/docker/`, `internal/app/`, `internal/tui/`, `internal/ui/styles/`, and `cmd/` -- with one-way dependency flow (`cmd -> app -> docker`)
  5. All Lipgloss styles are defined in a single `styles` package (zero inline style definitions in command files), and `--no-color` disables all styling via one call
  6. Unit tests achieve 90%+ coverage on business logic using the mocked Docker client
  7. Go version is aligned across `go.mod`, `.golangci.yml`, and all CI workflows
  8. All Cobra commands use `RunE` (not `Run` + `os.Exit`)
**Plans**: 4 plans

Plans:
- [ ] 02-01-PLAN.md -- Extract DockerService interface, concrete client, domain types, and mock into internal/docker/
- [ ] 02-02-PLAN.md -- Add per-operation context with timeouts, surface swallowed errors as warnings, switch to RunE
- [ ] 02-03-PLAN.md -- Centralize styles, move TUI models to internal/tui/, slim cmd/ to wiring, align Go versions
- [ ] 02-04-PLAN.md -- Unit test infrastructure with testify assertions and 90%+ coverage on business logic

**Key Risks**:
- Pitfall MP4: Wrong mock boundary -- use two-layer interface (low-level `DockerAPI` ~16 methods for transformation testing, high-level `DockerService` ~18 methods for business logic testing).
- Pitfall CP3: Goroutine leaks in watch mode from unbounded `fetchData()` without cancellation. Per-operation context with cancel-before-re-fetch pattern addresses this.
- Pitfall MP2: Race conditions. Enable `-race` flag in CI test runs.
- This is the largest phase (touches most files). Each migration step must preserve existing behavior. Run full test suite after each step.

---

### Phase 3: Core Container Operations and Safety System
**Goal**: Users can manage container lifecycles from the TUI with confidence -- every destructive action shows what will happen, explains reversibility, and re-checks state before executing
**Depends on**: Phase 2 (requires DockerService interface, per-operation context, centralized styles, package structure)
**Requirements**: R12, R13, R14, R15, R16
**Success Criteria** (what must be TRUE):
  1. User can start, stop, and restart a container from the TUI and see the updated state reflected immediately
  2. Before any destructive operation, a confirmation dialog shows: what will be affected, the severity tier (color-coded), whether the action is reversible, and how to undo it
  3. Deleting a container that was stopped when the user pressed "delete" but started running by the time confirmation completes results in a re-check and updated confirmation (not a silent `Force: true` deletion)
  4. User can run any destructive operation in dry-run mode to see what would happen without executing it
  5. User can delete networks from the analyze TUI (not silently ignored)
**Plans**: 2 plans

Plans:
- [ ] 03-01-PLAN.md -- Container lifecycle operations (start/stop/restart) with TUI keybindings, state refresh, and error handling
- [ ] 03-02-PLAN.md -- 5-tier safety confirmation system (informational/low-risk/moderate/high-risk/bulk-destructive) with TOCTOU protection and dry-run methods

**Key Risks**:
- Pitfall CP1 (top safety bug): TOCTOU race in delete operations. Current code uses `Force: true` unconditionally without re-checking container state. Must re-fetch state before every destructive op and never default to `Force: true`.
- Safety confirmation UX must be clear without being annoying. The 5-tier system (informational / low-risk / moderate / high-risk / bulk-destructive) should match user expectations.

---

### Phase 4: Viewing Features
**Goal**: Users can inspect container logs, see resource metrics, filter/search all list views, and pipe any command output to scripts via JSON
**Depends on**: Phase 2 (requires package structure, DockerService interface for log/stats APIs, centralized styles for --no-color), Phase 3 (safety system for any destructive actions in views)
**Requirements**: R17, R18, R19, R20, R28, R29, R30
**Success Criteria** (what must be TRUE):
  1. User can view historical logs for any container and follow live logs with automatic scrolling (ring buffer prevents unbounded memory growth)
  2. Container list shows per-container CPU and memory usage that updates in real-time
  3. User can press `/` in any list view to filter/search items, and the filter works across all visible columns
  4. Running any command with `--output-format json` produces valid, parseable JSON output (verifiable with `octo status --output-format json | jq .`)
  5. Running any command with `--no-color` produces output with zero ANSI escape sequences
  6. Slow operations (Docker API calls > 500ms) show a loading spinner instead of a frozen terminal
  7. Volume sizes in the analyze view reflect actual disk usage (from DiskUsage API, not the always-zero volume list API)
**Plans**: 3 plans in 1 wave

Plans:
- [ ] 04-01-PLAN.md — Container logs viewer with streaming, follow mode, ring buffer, search (simple text + regex), and export to ~/.octo/logs/
- [ ] 04-02-PLAN.md — Metrics display (CPU%, memory with trends), list filtering with smart detection, volume size fix via DiskUsage API, loading spinners
- [ ] 04-03-PLAN.md — JSON/YAML output with --output-format flag, fully qualified field names (containerId, containerName), --no-color working correctly

**Key Risks**:
- Pitfall MP5: Log streaming without backpressure can consume unbounded memory. Use a ring buffer with configurable max size (5000 lines default).
- Pitfall MP3: DiskUsage API can be slow on large systems. Cache results with 10-second TTL.
- Research flag: Verify Bubbles viewport API compatibility with Bubble Tea v1.3.4 before implementation.
- Research flag: Docker SDK `ContainerStats` CPU percentage calculation from delta values needs verification.

---

### Phase 5: Interactive Enhancements
**Goal**: Users can interact with the TUI using mouse, keyboard shortcuts, and number keys, and copy any displayed data to their clipboard
**Depends on**: Phase 4 (mouse and clipboard enhance the views built in Phase 4 -- logs, metrics, filtered lists)
**Requirements**: R21, R22, R23, R24
**Success Criteria** (what must be TRUE):
  1. User can click on any item in a TUI list to select it (mouse click-to-select works in all list views)
  2. User can navigate lists using arrow keys, number keys, or mouse clicks interchangeably
  3. User can press `y` to copy the currently selected item's details to the system clipboard, and this works on macOS (pbcopy), Linux X11 (xclip), Linux Wayland (wl-copy), and Windows (clip.exe) -- with a non-fatal fallback message if no clipboard tool is available
  4. User can generate shell completions for bash, zsh, and fish via `octo completion [shell]`
**Plans**: 2 plans in 2 waves

Plans:
- [x] 05-01-PLAN.md -- Mouse click-to-select in menu and analyze views, enable WithMouseCellMotion in all tea.NewProgram calls, mouse tests
- [x] 05-02-PLAN.md -- Clipboard copy with platform fallback chain (y key in analyze), shell completion command (bash/zsh/fish)

**Key Risks**:
- Pitfall MP1: Enabling mouse mode in Bubble Tea captures all mouse events, breaking terminal text selection. Use `WithMouseCellMotion()` (not `AllMotion`). Ship clipboard copy in the same phase as mouse support so users have an alternative copy mechanism.
- Research flag: Verify Bubble Tea v1.3.4 mouse API constants (`tea.MouseLeft`, `tea.MouseActionPress`) with `go doc` before implementation.
- Cross-platform clipboard testing is the time sink. Need to test on macOS, Linux (X11 + Wayland), WSL, SSH sessions, and tmux.

---

### Phase 6: Advanced Features
**Goal**: Users can open interactive shells inside running containers and manage Docker Compose projects as logical groups
**Depends on**: Phase 3 (requires container lifecycle and safety system), Phase 5 (benefits from stable TUI interaction patterns)
**Requirements**: R26, R27
**Success Criteria** (what must be TRUE):
  1. User can open an interactive shell (`/bin/sh` or `//bin/bash`) inside a running container from the TUI, with working stdin/stdout, proper terminal resizing (SIGWINCH), and clean terminal restoration when the shell exits
  2. Containers that belong to Docker Compose projects are visually grouped by project name in the container list (derived from `com.docker.compose.project` label)
  3. User can start/stop/restart all containers in a Compose project as a single action
**Plans**: 2 plans in 2 waves

Plans:
- [x] 06-01-PLAN.md -- Container exec/shell with PTY handling, SIGWINCH resize forwarding, tea.Exec integration, and terminal management
- [x] 06-02-PLAN.md -- Docker Compose awareness (label-based grouping, project-level lifecycle operations, visual TUI grouping)

**Key Risks**:
- Pitfall MP8 (highest complexity feature): Container exec requires exiting Bubble Tea, entering raw PTY mode via `golang.org/x/term`, forwarding SIGWINCH for terminal resizing, demuxing Docker exec streams, and cleanly restoring terminal state. Budget 2-3x estimated time. Study Docker CLI source (`cli/command/container/exec.go`).
- Pitfall MP9: Compose label fragility. Not all containers have Compose labels (manually created, Podman, etc.). Must gracefully degrade -- show ungrouped containers normally.
- Research flag: Windows ConPTY support for exec is unclear. May need to scope exec as macOS/Linux tier 1, Windows tier 2.
- Research flag: Verify `com.docker.compose.*` label names against a running Compose stack before implementation.

---

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Critical Bug Fixes and Terminal Safety | 1/1 | Complete | 2026-02-06 |
| 2. Architecture Foundation | 0/4 | Not started | - |
| 3. Core Container Operations and Safety System | 0/2 | Not started | - |
| 4. Viewing Features | 0/3 | Not started | - |
| 5. Interactive Enhancements | 2/2 | Complete | 2026-02-14 |
| 6. Advanced Features | 2/2 | Complete | 2026-02-14 |

## Coverage

| Requirement | Phase | Status |
|-------------|-------|--------|
| R01 | Phase 1 | Complete |
| R02 | Phase 1 | Complete |
| R03 | Phase 1 | Complete |
| R04 | Phase 1 | Complete |
| R05 | Phase 1 | Complete |
| R06 | Phase 2 | Pending |
| R07 | Phase 2 | Pending |
| R08 | Phase 2 | Pending |
| R09 | Phase 2 | Pending |
| R10 | Phase 2 | Pending |
| R11 | Phase 2 | Pending |
| R12 | Phase 3 | Pending |
| R13 | Phase 3 | Pending |
| R14 | Phase 3 | Pending |
| R15 | Phase 3 | Pending |
| R16 | Phase 3 | Pending |
| R17 | Phase 4 | Pending |
| R18 | Phase 4 | Pending |
| R19 | Phase 4 | Pending |
| R20 | Phase 4 | Pending |
| R21 | Phase 5 | Complete |
| R22 | Phase 5 | Complete |
| R23 | Phase 5 | Complete |
| R24 | Phase 5 | Complete |
| R25 | Phase 2 | Pending |
| R26 | Phase 6 | Complete |
| R27 | Phase 6 | Complete |
| R28 | Phase 4 | Pending |
| R29 | Phase 4 | Pending |
| R30 | Phase 4 | Pending |

**Mapped: 30/30** -- all v1 requirements assigned to exactly one phase.

---
*Created: 2026-02-06*
*Depth: Comprehensive (6 phases, 14 plans)*
