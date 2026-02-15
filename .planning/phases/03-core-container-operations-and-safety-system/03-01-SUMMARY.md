---
phase: 03-core-container-operations-and-safety-system
plan: 01
subsystem: docker, tui
tags: [container-lifecycle, start, stop, restart, bubbletea, keybindings]

# Dependency graph
requires:
  - phase: 02-architecture-foundation
    provides: DockerService interface, DockerAPI wrapper, MockDockerService, per-operation context/timeouts, centralized styles, analyze TUI model
provides:
  - StartContainer, StopContainer, RestartContainer methods in DockerService
  - TUI keybindings (s/t/r) for container lifecycle in analyze view
  - State refresh after lifecycle operations
  - Error handling with user-visible warnings
  - Unit tests for lifecycle operations (client + TUI)
affects: [03-02-safety-system, 06-01-container-exec, 06-02-compose-awareness]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Container lifecycle via DockerService interface methods"
    - "TUI keybinding -> tea.Cmd -> async Docker API call -> fetchResources() refresh"
    - "Error as DataMsg warning pattern (append to warnings slice, return via DataMsg)"

key-files:
  created: []
  modified:
    - internal/docker/interface.go
    - internal/docker/client.go
    - internal/docker/mock.go
    - internal/docker/timeout.go
    - internal/tui/analyze/model.go
    - internal/docker/client_test.go
    - internal/tui/analyze/model_test.go

key-decisions:
  - "Lifecycle methods accept caller-provided context (no context.Background() inside methods)"
  - "ContainerStop/Restart use default SDK timeout (nil pointer) -- no forced timeout override"
  - "No Force: true default -- Plan 03-02 adds confirmation dialogs for that decision"
  - "Menu model unchanged -- lifecycle operations only in analyze view where container selection exists"

patterns-established:
  - "Lifecycle operation pattern: key handler -> helper method -> tea.Cmd closure -> Docker API call -> fetchResources() on success / warning on error"
  - "canOperateOnSelected() guard: checks Selectable && !IsCategory for safe operation filtering"

# Metrics
duration: 5min
completed: 2026-02-07
---

# Phase 3 Plan 1: Container Lifecycle Operations Summary

**Container start/stop/restart via DockerService interface with s/t/r TUI keybindings and automatic state refresh**

## Performance

- **Duration:** 5 min (from STATE.md metrics)
- **Started:** 2026-02-07
- **Completed:** 2026-02-07
- **Tasks:** 5
- **Files modified:** 7

## Accomplishments
- Added StartContainer, StopContainer, RestartContainer to DockerService interface and Client implementation
- Added MockDockerService implementations with configurable Fn fields for all three lifecycle methods
- Added s/t/r keybindings in analyze TUI model with container-type guard checks
- Each lifecycle operation automatically re-fetches the container list to reflect updated state
- Errors from Docker API calls display as user-visible warnings in the TUI
- Unit tests cover success paths, error paths, and state transition scenarios

## Task Commits

The implementation was committed atomically:

1. **Tasks 1-5: Container lifecycle operations** - `bcabb90` (feat)
   - Interface methods, client implementations, mock methods, TUI keybindings, unit tests

## Files Created/Modified
- `internal/docker/interface.go` - Added StartContainer, StopContainer, RestartContainer to DockerService interface; ContainerStart, ContainerStop, ContainerRestart to DockerAPI
- `internal/docker/client.go` - Implemented lifecycle methods wrapping Docker SDK calls with proper context forwarding
- `internal/docker/mock.go` - Added StartContainerFn, StopContainerFn, RestartContainerFn fields and method implementations
- `internal/docker/timeout.go` - Added TimeoutAction constant for per-operation timeouts
- `internal/tui/analyze/model.go` - Added s/t/r key handlers, startSelectedContainer, stopSelectedContainer, restartSelectedContainer helper methods, canOperateOnSelected and selectedEntry helpers
- `internal/docker/client_test.go` - TestStartContainer, TestStopContainer, TestRestartContainer with success and error subtests
- `internal/tui/analyze/model_test.go` - TestAnalyze_StartContainer_Success, TestAnalyze_StopContainer_Error, TestAnalyze_RestartContainer_Success, TestAnalyze_CanOperateOnSelected

## Decisions Made
- Lifecycle methods accept caller-provided context.Context -- no internal context.Background() usage
- Docker SDK container.StopOptions and container.StopOptions for restart use default timeout (nil) rather than forcing a timeout
- No Force: true default on any operation -- Plan 03-02 adds confirmation dialogs for that decision
- Menu model intentionally unchanged: lifecycle operations only available in analyze view where per-container selection exists
- Error messages use DataMsg.Warnings pattern rather than a separate WarningMsg type (consistent with existing analyze model error handling)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- DockerService interface now has lifecycle methods, ready for Plan 03-02 (safety confirmation system)
- Mock implementations ready for testing confirmation dialogs
- TUI keybinding pattern established for future operations

## Self-Check: PASSED

All key files verified present on disk. Commit `bcabb90` verified in git history.

---
*Phase: 03-core-container-operations-and-safety-system*
*Completed: 2026-02-07*
