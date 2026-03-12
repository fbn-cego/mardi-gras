# Upstream Check — 2026-03-12

## TL;DR

Heavy merge day on Gas Town (~30 commits, 15+ merged PRs) and active Dolt hardening on Beads (~30 commits). No new releases on either repo — still waiting for bd v0.59.1+. Key GT merges: event-driven polecat lifecycle with `FIX_NEEDED` state (PR #2633), convoy `--base-branch` support (#2398), refinery merge strategy config (#2609), daemon pressure checks (#2370), and mail auto-nudge (#2674). GT PR #2672 confirms `--flat` flag removal — validates our same-day fix. Beads adds `--design-file` flag, global `~/.config/beads/PRIME.md` fallback, and `bd bootstrap` now auto-executes recovery actions.

## Current baseline

- mg version: v0.8.0
- Beads: v0.58.0 installed (`/usr/local/bin/bd`); v0.59.0 latest release (DO NOT UPGRADE)
- Gas Town: v0.11.0 latest release
- go.mod: charm.land/bubbletea/v2 v2.0.2, charm.land/lipgloss/v2 v2.0.2 (just upgraded)
- Previous check: [upstream-check-2026-03-10.md](upstream-check-2026-03-10.md) + journal entry 2026-03-11

## Breaking changes

### 1. `--flat` flag removed from bd (confirmed)

**GT PR #2672**: "move `--flat` fallback into `Beads.run` to prevent re-injection" — Gas Town itself was hitting the same `--flat` removal issue we encountered today. The flag no longer exists in bd; `--json` output is always flat.

**mg impact**: Already fixed today in commit `147f495` — removed `--flat` from `bdListArgs()` in `internal/data/source.go`.

**Action**: Done.

### 2. `bd doctor --clean --json` fixed (Beads main)

**Commit**: `2c24ce6` — fixes `doctor --clean --json` and blocks `--data-dir` in server mode.

**mg impact**: If mg calls `bd doctor --json`, the output is now more reliable. Low impact — our `bd doctor --agent` call in problems overlay doesn't use `--clean`.

**Action**: None required.

## Feature opportunities

### 1. Event-driven polecat lifecycle with FIX_NEEDED (GT #2633 — merged)

Polecats now have a `FIX_NEEDED` feedback loop state. When a polecat's work fails review, it enters `FIX_NEEDED` instead of being decommissioned. This is a new agent state mg should render.

**Effort**: Small | **Files**: `internal/ui/theme.go` (AgentStateColor), `internal/ui/symbols.go` (new symbol), `internal/views/gastown.go`

### 2. Convoy `--base-branch` support (GT #2398 — merged)

`gt convoy create` now accepts `--base-branch` to create convoys against non-default branches. mg's convoy create flow could expose this option.

**Effort**: Small | **Files**: `internal/gastown/convoy.go`

### 3. Refinery merge strategy config (GT #2609 — merged)

Refinery now supports `direct` vs `PR` merge modes via config. Informational — could surface in Gas Town panel's agent detail view.

**Effort**: Small | **Files**: `internal/views/gastown.go` (agent detail)

### 4. Daemon pressure checks (GT #2370 — merged)

Opt-in resource pressure checks before spawning agents. Could surface pressure warnings in problems overlay.

**Effort**: Medium | **Files**: `internal/gastown/problems.go`

### 5. `--design-file` flag (Beads main)

**Commits**: `71d7ae5`, `56cd587` — `bd create/update --design-file <path>` reads design from a file instead of inline. mg's create form could use this for multi-line design input.

**Effort**: Small | **Files**: `internal/components/create_form.go`

### 6. Global PRIME.md fallback (Beads main)

**Commit**: `3a2d351` — `~/.config/beads/PRIME.md` as global fallback for session priming. Informational — no mg change needed.

**Effort**: N/A

### 7. `bd bootstrap` auto-executes recovery (Beads main)

**Commit**: `efd7568` — `bd bootstrap` now runs recovery actions automatically instead of just printing advice. Could improve mg's error recovery UX when bd fails to load.

**Effort**: Medium | **Files**: `internal/data/source.go` (error recovery path)

### 8. Mail auto-nudge (GT #2674 — merged)

Agents are now auto-nudged to reply via mail rather than in chat. Informational — mg's nudge action (`n` key) still works via `gt nudge`.

**Effort**: N/A

### 9. `bd stdout warnings stripped` (GT)

**Commit**: `3164aad` — GT strips bd stdout warnings to prevent JSON parse corruption. Validates our approach in `internal/data/exec.go` where we handle mixed stdout.

**Effort**: N/A (confirms our approach is correct)

## Recommended actions

| # | Action | Priority | Effort | Status |
|---|--------|----------|--------|--------|
| 1 | Track bd v0.59.1+ release | critical | small | **WAITING** — still no release |
| 2 | Render `FIX_NEEDED` polecat state in Gas Town panel | medium | small | NEW |
| 3 | Surface epic progress (N/M) in detail panel | medium | medium | **BLOCKED** — requires bd v0.59.1+ |
| 4 | Surface `bd context` in header/footer | medium | small | **BLOCKED** — requires bd v0.59.1+ |
| 5 | Add convoy `--base-branch` option to create flow | low | small | NEW |
| 6 | Surface refinery merge strategy in agent detail | low | small | NEW |
| 7 | Verify `gt nudge` still works with auto-nudge changes | low | small | DEFERRED — manual test |

## Raw commit log (since 2026-03-11)

### Beads (30 commits)

```
df7ee40 2026-03-12 chore: remove duplicate CLIDir() from linter merge artifact
61ddbe3 2026-03-12 fix: bd dolt remote add/list/remove operate on wrong directory (GH#2306, GH#2311)
491242c 2026-03-12 fix: update cliDir() → CLIDir() after linter rename
2d3ce08 2026-03-12 fix: sync CLI remotes into SQL server on store open (GH#2315)
36a3ecd 2026-03-12 fix: create .beads/ dir in server mode with external BEADS_DOLT_* env vars (GH#2519)
491c276 2026-03-12 fix: use CLIDir for dolt remote operations instead of root Path (GH#2306, GH#2311)
9dc875a 2026-03-12 feat: add --list and --doc flags to bd help for CLI doc generation (GH#2527)
67d6786 2026-03-12 fix: use explicit DOLT_ADD to prevent config corruption from stale working set (GH#2455)
2c24ce6 2026-03-12 fix: block data-dir in server mode + fix doctor --clean --json (GH#2438)
f1e6073 2026-03-12 Merge PR #2529: fix(deps): update dolthub/driver
efd7568 2026-03-12 feat: bd bootstrap executes recovery actions instead of printing advice
e42d222 2026-03-12 Merge PR #2526: purge stale refs
45fcd51 2026-03-12 fix(nix): use go mod edit instead of sed for version patching
4c43d96 2026-03-12 fix(deps): update dolthub/driver
56cd587 2026-03-12 fix: typo in --design-file error message
96802bf 2026-03-12 Merge PR #2524: --design-file flag
13c2ecc 2026-03-12 Merge PR #2511: global prime fallback
555abde 2026-03-11 fix(deps): update dolthub/driver
a333dca 2026-03-11 fix(docs): purge stale bd sync/import/--branch refs
f7061f9 2026-03-11 fix(docs): remove stale --branch flag
71d7ae5 2026-03-11 feat: add --design-file flag for reading design from files
db967a0 2026-03-11 fix(deps): update x/term
3a2d351 2026-03-11 feat: support global ~/.config/beads/PRIME.md fallback (GH#2330)
69573b8 2026-03-11 fix: add test coverage for quoted label values with colons
f88c298 2026-03-11 fix: Dolt test suite stability
a7755e5 2026-03-11 fix: prevent bd init from creating DB on another project's Dolt server
c9998fc 2026-03-11 fix: TestDoltStoreDependencies uses same-type issues for blocks dep
```

### Gas Town (30 commits)

```
a0d5945 2026-03-12 fix: resolve CI lint and test failures on main
2f7270e 2026-03-12 docs: add MVGT integration guide
c7cfa2d 2026-03-12 fix: validate remote names in compactor-dog
d5b5d20 2026-03-12 fix: compactor-dog use DOLT_FETCH instead of DOLT_PULL
f9ce9fc 2026-03-12 fix: replace grep -P with literal tab for macOS compat
41e50cc 2026-03-12 fix: sync with DoltHub remote before/after compaction
62d4519 2026-03-12 fix: address PR #2430 review feedback
44fe386 2026-03-12 feat: add executable run.sh for compactor-dog plugin
1df1723 2026-03-12 docs: fix-merge polecat lifecycle audit
5978367 2026-03-12 feat: github-sheriff v2 — single API call, PR categorization
4ba154a 2026-03-12 Merge PR #2398: convoy --base-branch support
207f1a5 2026-03-12 Merge PR #2609: refinery merge_strategy config (direct vs PR mode)
3164aad 2026-03-12 fix: strip bd stdout warnings to prevent JSON parse corruption
3bfdcb4 2026-03-12 Merge PR #2413: inject Dolt server port into agent tmux sessions
f68b77e 2026-03-12 Merge PR #2658: cross-platform harness drift fixes
5dc606f 2026-03-12 Merge PR #2633: event-driven polecat lifecycle with FIX_NEEDED
ef1d4ad 2026-03-12 Merge PR #2627: CLI-side cross-database dependency resolution
a6b0da4 2026-03-12 Merge PR #2625: cross-database convoy deps for multi-rig towns
98a2d06 2026-03-12 Merge PR #2370: daemon pressure checks before agent spawns
92ab22d 2026-03-12 Merge PR #2466: gt plugin sync auto-deploy
ecc00df 2026-03-12 Merge PR #2421: session-hygiene to deterministic run.sh
2eebe50 2026-03-12 Merge PR #2674: mail auto-nudge agents to reply
ab1d955 2026-03-12 fix: route agent bead creation to wisps
6c68457 2026-03-12 Merge PR #2668: use rig name and correct DB prefix
74f19f8 2026-03-12 Merge PR #2676: --reference for submodule init in worktrees
a3de309 2026-03-12 Merge PR #2677: degrade wait-idle to queue
b163dd9 2026-03-12 Merge PR #2399: guard against bd v0.58.0 non-JSON output
8935a8e 2026-03-12 Merge PR #2670: restrict remote branch deletion to polecat branches
eb2f5ec 2026-03-12 Merge PR #2672: move --flat fallback into Beads.run
1f9e270 2026-03-12 Merge PR #2666: bump tar in gt-model-eval
```

### Open PRs of interest

**Gas Town:**
- #2679: `gt crew at` auto-detect rig from crew member name
- #2678: `gt done` skip close for gt:rig identity beads
- #2662: exec-wrapper plugin type
- #2449: Trust tier escalation engine
- #2438: Daytona remote sandbox execution for polecats
- #2358: Decouple Agent Execution from Presentation (ACP)
