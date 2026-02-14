---
phase: 05-interactive-enhancements
plan: 01
subsystem: tui
tags: [bubbletea, mouse, tui, click-to-select, bubble-tea-v1.3.4]

# Dependency graph
requires:
  - phase: 02-architecture-foundation
    provides: TUI model structure in internal/tui/{menu,analyze,status}/
provides:
  - Mouse click-to-select in menu and analyze TUI views
  - tea.WithMouseCellMotion() enabled in all three TUI programs
  - Mouse event infrastructure for clipboard copy (05-02)
affects: [05-02-clipboard-copy]

# Tech tracking
tech-stack:
  added: []
  patterns: [tea.MouseMsg handling with headerHeight computation, scroll-offset-aware click targeting]

key-files:
  created: []
  modified:
    - cmd/menu.go
    - cmd/analyze.go
    - cmd/status.go
    - internal/tui/menu/model.go
    - internal/tui/analyze/model.go
    - internal/tui/menu/model_test.go
    - internal/tui/analyze/model_test.go

key-decisions:
  - "Used tea.MouseActionPress + tea.MouseButtonLeft (not deprecated tea.MouseLeft) for Bubble Tea v1.3.4 API"
  - "Computed headerHeight in InitMsg handler based on Docker state to avoid View() side effects"
  - "Analyze view uses fixed 3-line header offset plus scroll offset for click targeting"

patterns-established:
  - "Mouse click handler pattern: check Action+Button, compute index from Y-headerHeight, validate bounds"
  - "Header height tracking: compute once in Update() when state changes, not in View()"

# Metrics
duration: 7min
completed: 2026-02-14
---

# Phase 5 Plan 01: Mouse Click-to-Select Summary

**Mouse click-to-select in menu and analyze TUI views using Bubble Tea v1.3.4 MouseMsg API with header-height-aware coordinate mapping**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-14T20:30:37Z
- **Completed:** 2026-02-14T20:38:14Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Mouse click-to-select working in menu view (left-click on item selects and executes action)
- Mouse click-to-select working in analyze view (left-click with scroll offset accounting)
- Mouse events enabled in status view for consistency (wheel scroll support)
- Comprehensive test coverage: 13 new test cases across both views
- Help footers updated to advertise click navigation

## Task Commits

Each task was committed atomically:

1. **Task 1: Add mouse event handling and enable mouse in tea.NewProgram()** - `6f0f06a` (feat)
2. **Task 2: Add tests for mouse click handling** - `ed5abd6` (test)

## Files Created/Modified
- `cmd/menu.go` - Added tea.WithMouseCellMotion() to tea.NewProgram()
- `cmd/analyze.go` - Added tea.WithMouseCellMotion() to tea.NewProgram()
- `cmd/status.go` - Added tea.WithMouseCellMotion() to tea.NewProgram()
- `internal/tui/menu/model.go` - Added headerHeight field, computeHeaderHeight(), MouseMsg handler, updated help footer
- `internal/tui/analyze/model.go` - Added MouseMsg handler with scroll offset and confirmation guard, updated help footer
- `internal/tui/menu/model_test.go` - 6 new mouse test functions covering valid clicks, out-of-bounds, wrong button, release, header variations
- `internal/tui/analyze/model_test.go` - 7 new mouse test functions covering valid clicks, scroll offset, confirmation block, categories, out-of-bounds, wrong button, release

## Decisions Made
- Used `tea.MouseActionPress` + `tea.MouseButtonLeft` (modern API) instead of deprecated `tea.MouseLeft` constant
- Computed `headerHeight` in `Update()` when `InitMsg` is received rather than in `View()`, since Bubble Tea models are value types and View() uses a value receiver
- Analyze view uses a fixed 3-line header (title + separator + blank) plus scroll offset for click coordinate mapping
- Clicks during delete confirmation dialog are blocked to prevent accidental selection changes

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `go test` hangs in sandboxed environment due to macOS code signing / sandbox restrictions; resolved by compiling test binaries separately with `go test -c` then running the compiled binaries directly

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Mouse infrastructure is in place for Plan 05-02 (clipboard copy) which needs mouse events already enabled
- All three TUI programs now have tea.WithMouseCellMotion() active
- Pattern established for mouse event handling that 05-02 can extend

## Self-Check: PASSED

All 8 files verified present. Both task commits (6f0f06a, ed5abd6) verified in git log.

---
*Phase: 05-interactive-enhancements*
*Completed: 2026-02-14*
