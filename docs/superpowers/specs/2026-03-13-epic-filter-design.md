# Epic Filter — Design Spec

## Overview

Add the ability to filter the parade list to show only issues belonging to a specific epic. Users can select an epic via a fuzzy-searchable picker (triggered by `e` keybinding or command palette), and the parade narrows to that epic and all its descendants.

## State Changes

### App Model (`internal/app/app.go`)

Add two fields:

- `epicFilter string` — the selected epic's issue ID (empty = no filter active)
- `epicPicking bool` — whether the epic picker overlay is currently open

### Epic Picker

Reuse the existing `Palette` component pattern — a fuzzy-searchable list populated with all issues where `IssueType == TypeEpic` from the current dataset.

Each entry displays epic ID as `Name` and title as `Desc`. On selection, use `m.palette.SelectedName()` to extract the epic ID (same pattern as formula picking).

When an epic filter is already active, prepend a "Clear epic filter" entry at the top of the list (with `Name` set to a sentinel like `""` to distinguish from real epics).

When no epics exist in the dataset, the picker opens with an empty list — the palette already handles this gracefully ("No matching commands").

### Trigger

- **Keybinding `e`**: Always opens the epic picker. When a filter is active, the "Clear epic filter" entry appears first for quick clearing. This also allows switching directly to a different epic without needing to clear first.
- **Palette action `ActionFilterByEpic`**: Opens the picker identically.

## Filtering Logic

### New Helper: `FilterByEpic`

Location: `internal/data/filter.go`

```go
func FilterByEpic(issues []Issue, epicID string) []Issue
```

Keep issues where:
- `issue.ID == epicID`, OR
- `strings.HasPrefix(issue.ID, epicID+".")`

If `epicID` is empty, return all issues unchanged.

This captures the epic itself and all descendants at any depth. The dot separator prevents false matches (e.g., filtering by `mg-007` won't match `mg-0070`).

### Integration into `rebuildParade()`

Apply epic filter early in the pipeline, before text filter and focus mode:

```
all issues
  → FilterByEpic (if epicFilter != "")
  → FilterIssuesWithHighlights (text filter)
  → FocusFilter (if focusMode)
  → GroupByParade
  → NewParadeWithData
```

This lets text filter and focus mode compose naturally on top of the epic-scoped set.

**Important**: The groups recalculation condition must be expanded. The existing condition:

```go
if m.filterInput.Value() != "" || m.focusMode {
```

Must become:

```go
if m.filterInput.Value() != "" || m.focusMode || m.epicFilter != "" {
```

This ensures header counts and parade grouping reflect the epic-filtered set.

### Data Reload Handling

When the file watcher triggers a data reload, check whether the epic identified by `epicFilter` still exists in the new dataset. If it does not, auto-clear `epicFilter` and show a toast notification ("Epic filter cleared — epic no longer exists").

## UI Changes

### Header Badge (`internal/components/header.go`)

Add `EpicFilter string` field to the `Header` struct.

When `EpicFilter != ""`, render a badge after the existing count badges:

```
epic:mg-007
```

Styled with the theme's epic color to distinguish it from other badges.

### Footer Bindings (`internal/components/footer.go`)

Add `e` to the parade keybinding hints (label: "epic").

### Palette Command (`internal/components/palette.go`)

Add `ActionFilterByEpic` to the action enum and a corresponding command:

```
Name: "Filter by epic"
Desc: "Show only issues under a specific epic"
Key:  "e"
Action: ActionFilterByEpic
```

Always included in `buildPaletteCommands()` (epics are a core issue type, not gated on Gas Town).

## Epic Picker Component

Reuse the `Palette` component with a custom command list. Build epic entries as:

```go
PaletteCommand{Name: epicID, Desc: epicTitle, Action: ActionFilterByEpic}
```

On selection, extract the epic ID via `m.palette.SelectedName()` (same pattern as formula picking in the existing codebase).

### Message Flow

1. User presses `e` (or selects palette action) → app sets `epicPicking = true`, builds epic command list, opens palette
2. User selects an epic → palette returns result, app reads `m.palette.SelectedName()` to get epic ID
3. If selected name is empty (the "Clear" entry) → clear `epicFilter`; otherwise set `epicFilter = selectedName`
4. App calls `rebuildParade()`

## Files Changed

| File | Change |
|------|--------|
| `internal/data/filter.go` | Add `FilterByEpic()` function |
| `internal/data/filter_test.go` | Tests for `FilterByEpic()` |
| `internal/app/app.go` | Add `epicFilter`/`epicPicking` state, `e` key handler, epic picker flow, integrate into `rebuildParade()`, expand groups condition, data reload handling |
| `internal/components/palette.go` | Add `ActionFilterByEpic` to action enum, add command to `buildPaletteCommands()` |
| `internal/components/header.go` | Add `EpicFilter` field, render epic filter badge |
| `internal/components/footer.go` | Add `e` to parade keybinding hints |

## Testing

- `FilterByEpic`: unit tests for exact match, prefix match at multiple depths, no match, empty epicID passthrough, no false prefix matches (e.g., `mg-007` vs `mg-0070`)
- Palette commands: verify `ActionFilterByEpic` is included in `buildPaletteCommands()` output
- Keybinding: verify `e` opens picker in both filter-active and filter-inactive states
- Data reload: verify epic filter auto-clears when the filtered epic is removed from the dataset
