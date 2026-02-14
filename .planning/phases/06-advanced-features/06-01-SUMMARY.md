---
phase: 06-advanced-features
plan: 01
subsystem: tui
tags: [docker-exec, pty, sigwinch, bubbletea, tea-exec, golang-x-term, interactive-shell]

# Dependency graph
requires:
  - phase: 02-architecture-foundation
    provides: Two-layer Docker interface (DockerAPI + DockerService)
  - phase: 05-interactive-enhancements
    provides: Stable TUI analyze view with key handlers
provides:
  - Interactive container shell via 'x' key in analyze TUI
  - DockerExecCommand implementing tea.ExecCommand
  - 5 exec methods on DockerAPI interface
  - API() accessor on DockerService
  - Platform-specific SIGWINCH resize handling
affects: [06-02-compose-awareness]

# Tech tracking
tech-stack:
  added: [golang.org/x/term]
  patterns: [tea.Exec terminal handoff, platform-specific build tags for signal handling]

key-files:
  created:
    - internal/docker/exec.go
    - internal/docker/resize_unix.go
    - internal/docker/resize_windows.go
  modified:
    - internal/docker/interface.go
    - internal/docker/client.go
    - internal/docker/mock.go
    - internal/docker/types.go
    - internal/docker/timeout.go
    - internal/tui/analyze/model.go
    - internal/tui/common/messages.go

key-decisions:
  - "DockerExecCommand takes DockerAPI directly, not DockerService -- exec needs raw SDK calls"
  - "API() accessor on DockerService to expose DockerAPI for exec without breaking abstraction"
  - "Platform-specific build tags only for SIGWINCH signal registration, not entire exec logic"
  - "No timeout on exec sessions -- interactive sessions have no predictable duration"

patterns-established:
  - "tea.Exec pattern: create ExecCommand struct, pass to tea.Exec with callback for ExecFinishedMsg"
  - "Platform-specific signal handling via build-tagged files (resize_unix.go, resize_windows.go)"

# Metrics
duration: 5min
completed: 2026-02-14
---

# Phase 6 Plan 1: Container Exec/Shell Summary

**Interactive container shell via tea.Exec with PTY raw mode, SIGWINCH resize forwarding, and clean TUI restoration**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-14T22:43:02Z
- **Completed:** 2026-02-14T22:47:50Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Interactive shell sessions inside running containers via 'x' key in analyze TUI
- Full PTY handling with terminal raw mode and bidirectional I/O streaming
- SIGWINCH resize forwarding so terminal size changes propagate to container
- Clean terminal restoration after shell exit with automatic TUI data refresh
- Error handling for non-running containers (shows status message, no crash)

## Task Commits

Each task was committed atomically:

1. **Task 1: Docker exec infrastructure** - `a9e6ade` (feat)
2. **Task 2: DockerExecCommand, SIGWINCH handler, and TUI integration** - `433f18a` (feat)

## Files Created/Modified
- `internal/docker/exec.go` - DockerExecCommand implementing tea.ExecCommand with PTY and I/O streaming
- `internal/docker/resize_unix.go` - SIGWINCH signal registration for Unix
- `internal/docker/resize_windows.go` - No-op signal registration stub for Windows
- `internal/docker/interface.go` - 5 exec methods on DockerAPI, API() on DockerService
- `internal/docker/client.go` - API() accessor implementation
- `internal/docker/mock.go` - Mock implementations for all new interface methods
- `internal/docker/types.go` - ExecOptions type
- `internal/docker/timeout.go` - TimeoutExecCreate constant
- `internal/tui/analyze/model.go` - 'x' key handler, canExecOnSelected(), ExecFinishedMsg handler
- `internal/tui/common/messages.go` - ExecFinishedMsg type
- `go.mod` / `go.sum` - golang.org/x/term dependency

## Decisions Made
- Used DockerAPI directly in DockerExecCommand rather than going through DockerService, because exec needs raw SDK calls (ContainerExecCreate, Attach, Resize) that don't fit the domain-level service abstraction
- Added API() accessor to DockerService to let TUI code access DockerAPI without breaking the interface boundary
- Split platform-specific code into build-tagged files only for SIGWINCH registration (resize_unix.go/resize_windows.go), keeping the rest of exec.go platform-independent
- No timeout on exec sessions since they are interactive with no predictable duration; only exec create uses TimeoutExecCreate (10s)
- Used /bin/sh as default shell (available in all containers, unlike /bin/bash)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed go.mod golang.org/x/term as direct dependency**
- **Found during:** Task 2
- **Issue:** After `go get golang.org/x/term` in Task 1, it was marked `// indirect`. After exec.go imported it, `go mod tidy` was needed to promote it to direct.
- **Fix:** Ran `go mod tidy` after creating exec.go
- **Files modified:** go.mod, go.sum
- **Verification:** `go build ./...` succeeds, go.mod shows `golang.org/x/term v0.40.0` without indirect marker
- **Committed in:** 433f18a (Task 2 commit)

**2. [Rule 1 - Bug] Fixed formatting alignment in mock.go and timeout.go**
- **Found during:** Task 2 (during `make fmt`)
- **Issue:** Adding new fields to MockDockerService/MockDockerAPI structs and timeout constants changed tab alignment for existing entries
- **Fix:** Ran `make fmt` to apply standard Go formatting
- **Files modified:** internal/docker/mock.go, internal/docker/timeout.go
- **Verification:** `make fmt-check` passes
- **Committed in:** 433f18a (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 formatting)
**Impact on plan:** Both auto-fixes necessary for correct build and formatting compliance. No scope creep.

## Issues Encountered
- golangci-lint not installed in the environment (reported by `make lint`), so lint verification relied on `go vet` and `go build` which both pass cleanly
- `go test` failed with sandbox permission error (operation not permitted), not a code issue -- verified correctness via `go build` and `go vet` instead

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Exec/shell feature complete, ready for manual testing with running Docker containers
- Phase 6 Plan 2 (compose awareness) can proceed independently
- Phase 4 blocker (golangci-lint formatter) was not triggered in this plan since all interface methods have callers

## Self-Check: PASSED

- All created files exist (exec.go, resize_unix.go, resize_windows.go, 06-01-SUMMARY.md)
- All commits found (a9e6ade, 433f18a)
- All packages compile (docker, tui, cmd)
- go vet passes with no issues

---
*Phase: 06-advanced-features*
*Completed: 2026-02-14*
