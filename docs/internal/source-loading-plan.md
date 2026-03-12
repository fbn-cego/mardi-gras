# Source Loading Plan

Date: 2026-03-12
Issue: `mg-e6k`

## Current State

Mardi Gras now treats the Beads CLI as the primary runtime source of issue data:

- `cmd/mg/main.go` prefers `SourceCLI` whenever `.beads/` exists and `bd` is on `PATH`
- `internal/data/source.go` loads CLI data with `bd list --json --flat --limit 0 --all`
- `internal/data/source.go` enriches selected issues with `bd show --long --json`
- all mutations already go through `bd` CLI
- JSONL mode remains only for explicit `--path`, tests, sample fixtures, and legacy workspaces where `bd` is unavailable

This is the correct direction. It matches upstream reality and the way mg is already used in active Beads and Gas Town workspaces.

## Decision

We should treat JSONL support as legacy compatibility, not as a first-class product path.

That means:

- new source-loading features should target CLI mode first
- we should not add new product capabilities that depend on JSONL-only behavior
- JSONL support should remain available only where it still provides real compatibility value

## Why

The upstream direction is consistent:

- JSONL has already been removed or downgraded multiple times upstream
- current Beads workflows are Dolt/CLI-centric, not file-centric
- mg already had to add CLI mode to stay compatible with modern Beads
- the highest-risk data-loading failures we have seen recently were CLI routing problems, not JSONL parsing problems

Operationally, this means our real runtime risk is now CLI correctness and workspace identity, not file parsing.

## Planned Direction

### Phase 1: Present (keep dual-source, CLI-preferred)

Status: active now

- keep `SourceCLI` as the preferred runtime path
- keep JSONL explicit support via `--path`
- keep legacy fallback to JSONL only when `bd` is unavailable but `.beads/issues.jsonl` exists
- continue hardening CLI mode (`--flat`, prefix sanity checks, fake-`bd` integration tests)

This preserves compatibility without pretending the two paths are equally strategic.

### Phase 2: Legacy containment

Status: next docs/UX pass

- label JSONL mode as legacy in user-facing docs
- avoid feature work that only improves JSONL mode unless it fixes breakage
- keep tests and fixtures that exercise JSONL parsing, because they are still useful for sample/demo runs and explicit file-path usage
- if user confusion grows, consider an explicit `--source=cli|jsonl` override rather than continuing to broaden auto-detection rules

### Phase 3: Optional runtime simplification

Status: deferred

Revisit only if these conditions hold:

- supported Beads baseline reliably includes the CLI features mg depends on
- we no longer need implicit JSONL fallback for real users
- sample/demo/test workflows have an acceptable replacement or an explicit JSONL-only mode

At that point we can evaluate:

- removing implicit JSONL auto-detection from normal startup
- keeping JSONL only behind `--path`
- or fully removing runtime JSONL support while retaining fixture parsing in tests/tools

This should be a deliberate product decision, not a silent code cleanup.

## Compatibility Strategy

What we keep:

- `LoadIssues()` and JSONL parsing for explicit file usage
- `WatchFile()` for JSONL mode
- sample fixtures in `testdata/`
- tests that verify `resolveSource()` still behaves correctly for legacy environments

What we stop investing in:

- JSONL-first roadmap assumptions
- new UI or behavior that relies on JSONL being the primary or canonical runtime source
- docs that describe the app as fundamentally file-watcher-driven when CLI mode is now the common path

## Concrete Follow-ups

Short-term:

- keep `mg-2qa` open until `bd context` exists in the supported local baseline
- update older internal docs that still describe the roadmap as JSONL-first
- ensure architecture docs consistently mention `--flat` on CLI fetches

Longer-term:

- decide whether `--path` remains a permanent escape hatch
- decide whether direct Dolt access is worth the complexity once CLI mode is stable enough

## Code Touchpoints

- `cmd/mg/main.go`
- `cmd/mg/main_test.go`
- `internal/data/source.go`
- `internal/data/loader.go`
- `internal/data/watcher.go`
- `internal/components/footer.go`
- `docs/ARCHITECTURE.md`

## Non-Goals

- removing JSONL support immediately
- switching to direct Dolt in this planning issue
- changing the current startup behavior for legacy users in this planning issue
