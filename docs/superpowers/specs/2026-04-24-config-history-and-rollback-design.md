# Config History And Rollback Design

## Summary

Add a durable configuration history feature for management-driven `config.yaml` writes.
The system stores the latest 8 successful configuration revisions on the backend, exposes
history through management APIs, and allows operators to preview diffs and restore an
older revision from the config panel.

This design intentionally stays YAML-first. The current product already treats
`config.yaml` as the source of truth across backend handlers and the frontend config
panel. The history feature extends that model instead of introducing a database-backed
audit subsystem.

## User Decisions Locked In

- Every management API path that persists `config.yaml` must create history.
- Restore requires diff preview and explicit second confirmation.
- A successful restore creates a new latest version.
- If the config page has unsaved local edits, restore is blocked.
- The history list shows timestamp, simple incremental version number, and diff summary.
- Version numbers use a simple sequence such as `#128`.

## Goals

- Preserve the last 8 successful config revisions on the backend.
- Keep history generation consistent across all management API write paths.
- Let operators inspect revision summaries and full diffs before restoring.
- Make restore safe, explicit, and reversible.
- Reuse the existing YAML-oriented config architecture and UI patterns.

## Non-Goals

- Do not build a general audit log with actor identity, IP attribution, or free-form notes.
- Do not track direct out-of-band file edits that bypass the management API.
- Do not add database storage for revision history.
- Do not redesign the config page information architecture or visual language.

## Constraints

- `config.yaml` remains the only source of truth for runtime configuration.
- History must survive process restarts.
- Restore must use the same validated write path as normal saves.
- Failed writes must not create history entries.
- History retention is capped at 8 revisions.
- The feature must work for both raw `/config.yaml` writes and field-level management API writes
  that call `persist()` / `persistLocked()`.

## Why File-Based History

The existing management config flow is file-based and YAML-first. Introducing a database
layer for 8 retained revisions would add operational and code complexity without solving a
real product problem.

The recommended storage model is a hidden sibling directory next to `config.yaml`:

- `<config-dir>/.config-history/000001.yaml`
- `<config-dir>/.config-history/000001.meta.json`

This keeps the implementation debuggable by humans, avoids schema migration, and allows
recovery with standard filesystem tools if needed.

## Backend Architecture

### New Package Responsibility

Add a small history service under `internal/managementasset/`, for example
`config_history.go`, because this package already owns local management asset persistence
concerns. The new service owns:

- revision file layout
- version allocation
- summary generation
- retention cleanup
- restore file loading
- unified config save with history

The service must not change runtime config semantics. It only wraps validated config
write behavior with revision bookkeeping.

### Storage Layout

For a config file at:

`/path/to/config.yaml`

store history in:

`/path/to/.config-history/`

Each revision uses two files:

- `000001.yaml`
- `000001.meta.json`

The zero-padded filename keeps lexical sorting aligned with version order and removes the
need for a separate index file.

### Revision Metadata

Each `meta.json` file contains:

```json
{
  "version": 1,
  "created_at": "2026-04-24T16:32:10Z",
  "summary": [
    "routing.strategy",
    "codex-api-key[2].proxy-url"
  ],
  "action": "save",
  "restored_from": null
}
```

Rules:

- `version` is monotonically increasing and never reused.
- `action` is `save` or `restore`.
- `restored_from` is populated only for restore-generated revisions.
- `summary` contains up to 3 concise diff path entries.

### Unified Save Path

Add a single backend entry point such as:

`SaveConfigWithHistory(configPath string, nextYAML []byte, action RevisionAction, restoredFrom *int) error`

Responsibilities:

1. Read the current `config.yaml` raw bytes.
2. If the raw content is identical to `nextYAML`, exit without creating a revision.
3. Validate the candidate config through the existing config load path.
4. Persist the new `config.yaml`.
5. Reload the runtime config.
6. Generate diff summary between old YAML and new YAML.
7. Allocate the next version number.
8. Write revision YAML and metadata files.
9. Delete revisions older than the newest 8.

This function becomes the only path allowed to persist config content from management
handlers.

### Existing Call Sites To Rewire

Two existing write paths must be redirected:

1. Raw config save:
   - `internal/api/handlers/management/config_basic.go`
   - `PutConfigYAML`

2. Structured field saves:
   - `internal/api/handlers/management/handler.go`
   - `persist()`
   - `persistLocked()`

That guarantees the product choice of “all management API writes generate history”.

### Summary Generation

Summaries should be semantic, not raw line snippets.

Algorithm:

1. Parse old and new YAML into generic structures.
2. Recursively compare them.
3. Emit changed paths in dot/bracket notation.
4. Keep the first 3 meaningful paths after stable sorting.
5. If no structural difference is available but raw YAML changed, fall back to:
   - `raw yaml changed`

Examples:

- `routing.strategy`
- `payload.override[1].models`
- `openai-compatibility[0].api-key-entries[1].proxy-url`

This keeps the list compact while still useful for human scanning.

### Restore Semantics

Restore is not a direct file overwrite shortcut.

Instead:

1. Read revision `#N` YAML from `.config-history`.
2. Feed that YAML into `SaveConfigWithHistory(...)` with:
   - `action=restore`
   - `restored_from=N`
3. Return:
   - restored source version
   - newly created current version

Example:

- Current config is `#128`
- User restores `#123`
- Backend writes config content from `#123`
- Backend creates new revision `#129` with `action=restore` and `restored_from=123`

This preserves a complete forward-only history.

### Concurrency And Safety

- Reuse the existing handler mutex around config persistence.
- Version allocation reads the max existing version and writes `max + 1`.
- Revision files must be written atomically.
- If revision file write fails after config persistence succeeds, return an error and log it as
  a persistence inconsistency. Do not silently claim success.
- Retention cleanup must run after the new revision is durably written.

## Management API Contract

### `GET /config-history`

Returns the latest 8 revisions in descending version order.

Response shape:

```json
{
  "items": [
    {
      "version": 128,
      "created_at": "2026-04-24T16:32:10Z",
      "summary": ["routing.strategy", "codex-api-key[2].proxy-url"],
      "action": "save",
      "restored_from": null
    }
  ]
}
```

Notes:

- This endpoint returns metadata only.
- The current live config is not represented as a separate pseudo-history entry.

### `GET /config-history/:version`

Returns metadata plus full YAML content for one revision.

Response shape:

```json
{
  "version": 123,
  "created_at": "2026-04-21T10:00:00Z",
  "summary": ["payload.override[1].models"],
  "action": "save",
  "restored_from": null,
  "yaml_content": "host: 0.0.0.0\n..."
}
```

### `POST /config-history/:version/restore`

Restores a revision through the same validated save path.

Response shape:

```json
{
  "status": "ok",
  "restored_from": 123,
  "new_version": 129,
  "changed": ["config"]
}
```

Failure cases:

- `404` if the revision does not exist
- `422` if the stored YAML is no longer valid under current validation rules
- `500` for persistence failures

## Frontend Design

### Entry Point

Reuse the existing config page at:

- `Cli-Proxy-API-Management-Center/src/pages/ConfigPage.tsx`

Add a `History` button next to the current floating config actions. Do not create a new page.
History is part of the config editing workflow, not a separate navigation destination.

### History Modal Layout

Open a large modal with two columns:

- left: revision list
- right: diff preview for the selected revision

This keeps operators in the current editing context while browsing history.

### Revision List Item

Each row shows:

- `#128`
- timestamp
- up to 3 summary lines or chips

List order is descending by version.

### Diff Preview

Do not create a second diff system.

Reuse the existing config diff utilities and visual presentation patterns:

- `src/components/config/DiffModal.tsx`
- `src/components/config/diffModalUtils.ts`

The history modal uses the current server YAML as the left side and the selected revision YAML
as the right side.

### Restore Flow

1. User opens history modal.
2. Frontend fetches `GET /config-history`.
3. User selects a revision.
4. Frontend fetches `GET /config-history/:version`.
5. Frontend renders diff against the current server config.
6. User clicks `Restore This Version`.
7. Frontend opens a second confirmation dialog with the same diff context.
8. On confirm, frontend calls `POST /config-history/:version/restore`.
9. On success, frontend reloads:
   - current `config.yaml`
   - visual/source state
   - history list
10. Frontend shows a notification like:
    - `Restored #123 and created #129`

### Unsaved Draft Rule

If the config page has unsaved local edits:

- history browsing remains allowed
- restore action is disabled
- a warning banner is shown inside the history modal

Recommended copy:

`You have unsaved local changes. Save or discard them before restoring a historical revision.`

This prevents accidental overwrite of local draft state while still allowing inspection.

### Empty State

If no revisions exist yet, show an explicit empty state inside the modal.
Do not hide the entry point.

## State Model

Suggested frontend state additions:

- `historyModalOpen`
- `historyListLoading`
- `historyItems`
- `selectedHistoryVersion`
- `historyDetailLoading`
- `selectedHistoryYaml`
- `restoreConfirmOpen`
- `restoreLoading`

These are enough to drive the feature without creating a new page-level state subsystem.

## Error Handling

### Backend

- Failed config validation returns `422` and does not create a revision.
- Failed restore returns an error and does not create a revision.
- Failed history list/detail reads return standard management API errors.
- Revision file corruption should be logged with context and excluded from list responses
  instead of crashing the handler.

### Frontend

- If list load fails, show an inline error state inside the history modal.
- If detail load fails, keep the list usable and show an error in the preview pane.
- If restore fails, keep the modal open and preserve the current local editor state.
- A failed restore must never silently reset the current editing session.

## Testing Plan

### Backend Unit Tests

- first successful save creates revision `#1`
- repeated identical save creates no new revision
- more than 8 saves retains only the newest 8
- `persistLocked()` path creates revisions
- raw `PUT /config.yaml` path creates revisions
- restore creates a new latest revision with `action=restore`
- restore failure creates no new revision
- corrupted revision metadata is skipped safely

### Backend Handler Tests

- `GET /config-history` returns descending versions
- `GET /config-history/:version` returns `404` for missing revision
- `POST /config-history/:version/restore` updates runtime config and returns the new version

### Frontend Tests

- history button opens the modal
- revision list renders timestamp, version, and summary
- empty history state renders correctly
- selecting a revision loads and displays diff preview
- restore button is disabled when the page has unsaved edits
- successful restore reloads config and clears dirty state
- failed restore keeps current editor state intact

### Verification Commands

Backend:

```bash
gofmt -w .
go test ./...
go build -o test-output ./cmd/server && rm test-output
```

Frontend:

```bash
npm run type-check
npm run build
```

## Rollout Plan

1. Implement backend history service and tests.
2. Rewire both config persistence paths to the unified history save function.
3. Add management history APIs and tests.
4. Add frontend history modal and restore flow.
5. Verify restore behavior against real config reload flow.

## Rollback Plan

If this feature misbehaves after rollout:

1. Disable the frontend history button by removing the entry point.
2. Revert backend persistence rewiring to the old save path.
3. Leave `.config-history/` files on disk; they are harmless and can be inspected manually.

This keeps rollback simple and avoids deleting potentially valuable revision data during
incident response.
