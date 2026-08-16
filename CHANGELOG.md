# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 16-08-2026

### New Features

- Thumbnail repair tools: vault-wide "Repair all thumbnails" and orphan cleanup, both from the settings modal
- Thumbnails now self-heal on demand when missing or stale
- DB migrations are now versioned (schema_migrations)

### Fixes

- Debounced Name/Notes/Link edits could save to the wrong file when switching selection quickly; pending edits now flush to the correct file on switch
- Fixed a data race on vault state
- Search queries parameterized; FTS/LIKE escaping fixed
- Fixed malformed thumbnail URL (busted `&` vs `?`)
- CancelThumbnailGeneration now actually cancels the worker pool
- Removed fake client-side timeouts on file loading calls
- Rescan no longer leaves orphaned file_tags rows
- Category tags can no longer be applied via batch operations
- HTTP middleware no longer leaked as a Wails binding

### Performance

- Thumbnails served via HTTP endpoint only (removed base64 over JSON-RPC)
- File size stored in DB and sorted in-DB instead of stat-ing the vault
- Cached file count per vault call
- Batch ops use a bare-ID query instead of full file rows

### Changed

- Removed dead code; refactored thumbnail pipeline and HTTP middleware
- Frontend cleanup: typed getters, shared logger (FE + Go), modal extraction
- App version derived from wails.json at build time

### CI & Tooling

- GitHub Actions workflow: Go build/vet/test + frontend build
- Added linting: golangci-lint (Go) and ESLint/Prettier (frontend), wired into CI
- Added tests for scanner, file utils, and import

## [0.4.0] - 17-06-2026

### New Features

- Drag & Drop files into the vault folder (selected subfolder) with copy or move options
  - Drag files from Explorer onto the app window to import them into the vault
  - Choose whether to copy (keep original) or move (relocate into vault) the file
- add "Is Category" checkbox to tag creation/editing
  - Category tags can only be used as parents for other tags and cannot be assigned to files.

### Fixes

- Resolve relative paths for copy file path, copy folder path, delete original file
  The migration to relative paths `4f3f040` left several operations using DB-relative paths directly instead of resolving to absolute

## [0.3.0] - 15-06-2026

### New Features

Nested tag filtering — Parent tags now show all items from self + descendant tags, while leaf tags show only directly tagged items. `Ctrl+Click` accumulates multiple tags with AND logic.

### Fixes

- useSearch — Replaced loadFiles() with reloadFiles() so the user-selected sort order is preserved after clearing the search query.

### Changed

- Switch from absolute to relative paths for better portability
- Small UI changes:
  - Adjusted height so the panel fits inside `app-body`
  - Tag chip - Unified vertical size
  - Metadata — Filenames longer than 40 characters are truncated with an ellipsis; full name shown on hover via title tooltip
  - VaultSettingsModal — Now displays the total files indexed count in the vault.

### Dependencies

TypeScript 4.9.5 → 5.9.3

vue-tsc 1.8.27 → 2.x

Vite 3.0.7 → 5.4.21

@vitejs/plugin-vue 3.0.3 → 5.x

Vue 3.2.37 → 3.5.x

## [0.2.0] - 08-06-2026

### Added

- Initial public release
