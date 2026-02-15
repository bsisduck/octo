---
phase: 03-core-container-operations-and-safety-system
plan: 02
subsystem: docker, tui, ui
tags: [safety-tier, toctou, dry-run, confirmation-dialog, destructive-operations, network-deletion]

# Dependency graph
requires:
  - phase: 03-core-container-operations-and-safety-system
    provides: StartContainer, StopContainer, RestartContainer lifecycle methods, analyze TUI keybinding pattern
  - phase: 02-architecture-foundation
    provides: DockerService interface, DockerAPI wrapper, MockDockerService, centralized styles, analyze TUI model
provides:
  - 5-tier SafetyTier enum (Informational, LowRisk, Moderate, HighRisk, BulkDestructive)
  - ConfirmationInfo struct for destructive operation previews
  - 8 DryRun methods (RemoveContainer/Image/Volume/Network + PruneContainers/Images/Volumes/Networks)
  - TOCTOU protection (re-check state before confirmation AND before execution)
  - Tier-colored confirmation dialog in analyze TUI
  - RemoveNetwork implementation
  - Network deletion support in analyze view
affects: [04-viewing-features, 05-interactive-enhancements, 06-advanced-features]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "DryRun pattern: fetch current state, compute SafetyTier, return ConfirmationInfo without executing"
    - "TOCTOU two-phase re-check: Phase 1 (DryRun before confirmation) + Phase 2 (DryRun before execution, abort if tier changed)"
    - "Confirmation dialog rendering with TierStyle() color-coded styling"
    - "ConfirmationMsg tea.Msg type for async DryRun result delivery"

key-files:
  created: []
  modified:
    - internal/docker/types.go
    - internal/docker/interface.go
    - internal/docker/client.go
    - internal/docker/mock.go
    - internal/ui/styles/theme.go
    - internal/tui/analyze/model.go
    - internal/docker/client_test.go

key-decisions:
  - "TOCTOU protection via two-phase re-check: before confirmation AND before execution"
  - "Never default to Force: true; always re-fetch state before destructive operations"
  - "TierStyle takes int parameter (not docker.SafetyTier) to avoid circular import between styles and docker packages"
  - "Network deletion shows different confirmation than container (no reversibility/undo info)"

patterns-established:
  - "DryRun pattern: every destructive method has a *DryRun counterpart returning ConfirmationInfo"
  - "Two-phase deletion flow: 'd' key -> DryRun -> show confirmation -> 'y' key -> DryRun again -> verify tier unchanged -> execute"
  - "Safety abort on state change: if tier changes between confirmation and execution, abort with warning"

# Metrics
duration: 5min
completed: 2026-02-07
---

# Phase 3 Plan 2: Safety Confirmation System Summary

**5-tier safety system with TOCTOU protection, 8 DryRun methods, and tier-colored confirmation dialogs for all destructive Docker operations**

## Performance

- **Duration:** 5 min (from STATE.md metrics)
- **Started:** 2026-02-07
- **Completed:** 2026-02-07
- **Tasks:** 6
- **Files modified:** 7

## Accomplishments
- Defined SafetyTier enum with 5 severity levels and ConfirmationInfo struct with tier, title, description, resources, reversibility, undo instructions, and warnings
- Added 8 DryRun methods to DockerService interface and Client implementation covering all destructive operations (container/image/volume/network remove + prune)
- Implemented TOCTOU protection: RemoveContainer re-fetches container state before deletion; TUI calls DryRun twice (before confirmation dialog, and again before executing)
- Built tier-colored confirmation dialog in analyze TUI with resource details, reversibility status, undo instructions, and y/n prompt
- Added network deletion support in analyze view (RemoveNetwork implementation + DryRun + TUI switch cases)
- Added 5 tier-specific color styles to theme with TierStyle() helper function
- Added 8 mock DryRun method fields with nil-check implementations
- Added 6+ unit tests covering DryRun tier calculation, TOCTOU state change detection, and confirmation dialog rendering

## Task Commits

The implementation was committed atomically across 3 commits:

1. **Tasks 1-2, 6: SafetyTier, DryRun methods, mocks** - `a19c481` (feat)
   - SafetyTier enum, ConfirmationInfo struct, 8 DryRun methods, tier styles, mock implementations, RemoveNetwork
2. **Tasks 3-4: TOCTOU protection and confirmation dialog** - `7a53c9a` (feat)
   - TOCTOU re-check in RemoveContainer, two-phase deletion flow, renderConfirmationDialog, network deletion in TUI
3. **Task 5: Unit tests** - `8e5b388` (test)
   - 6 client tests (DryRun tiers, TOCTOU), analyze model tests (dialog rendering, delete flow)

## Files Created/Modified
- `internal/docker/types.go` - SafetyTier enum (5 levels with String()), ConfirmationInfo struct with tier/title/description/resources/reversible/undo/warnings
- `internal/docker/interface.go` - 8 DryRun methods added to DockerService interface
- `internal/docker/client.go` - 8 DryRun method implementations (state fetch + tier computation), TOCTOU re-check in RemoveContainer, RemoveNetwork implementation, formatBytes helper
- `internal/docker/mock.go` - 8 DryRunFn fields and method implementations with nil-check defaults
- `internal/ui/styles/theme.go` - TierInformationalStyle through TierBulkDestructiveStyle color definitions, ConfirmationBox/ConfirmYes/ConfirmNo styles, TierStyle() helper
- `internal/tui/analyze/model.go` - deleteConfirmInfo field, ConfirmationMsg type, reCheckAndShowConfirmation (Phase 1), deleteResource (Phase 2 with TOCTOU abort), renderConfirmationDialog, network deletion switch cases
- `internal/docker/client_test.go` - TestRemoveContainerDryRun_StoppedContainer, TestRemoveContainerDryRun_RunningContainer, TestRemoveContainerTOCTOU_StateChangeDuringDelete, TestRemoveImageDryRun, TestRemoveVolumeDryRun, TestPruneContainersDryRun

## Decisions Made
- TOCTOU protection implemented as two-phase re-check: DryRun before showing confirmation dialog AND DryRun before executing deletion; abort if tier changed between phases
- Never default to Force: true -- RemoveContainer checks if container is running and only forces if the user confirmed at the Moderate tier
- TierStyle() function takes int parameter instead of docker.SafetyTier to avoid circular import (styles package cannot import docker package)
- Network deletion intentionally shows different confirmation than container: no reversibility info since networks are trivially recreatable
- Prune operations compute tier based on resource count (0 = Informational, >0 = BulkDestructive)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Safety confirmation system complete: all destructive operations protected by DryRun + TOCTOU + confirmation dialog
- Phase 3 fully complete: container lifecycle (03-01) + safety system (03-02)
- Ready for Phase 4 (viewing features) when linter blocker is resolved
- DryRun pattern established for any future destructive operations

## Self-Check: PASSED

All 7 key files verified present on disk. All 3 commits (a19c481, 7a53c9a, 8e5b388) verified in git history.

---
*Phase: 03-core-container-operations-and-safety-system*
*Completed: 2026-02-07*
