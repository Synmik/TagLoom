# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.0] - 10-10-2026

### New Features

- PDFs and EPUBs are now scanned and thumbnailed: the first PDF page is rasterized by PDFium compiled to
  WebAssembly (`klippa-app/go-pdfium` + wazero — MIT/Apache, no CGO, no extra DLLs to ship), and an EPUB's cover
  is read straight out of the zip with the standard library (`archive/zip` + `encoding/xml`), looking at
  `cover-image`, then `<meta name="cover">`, then a cover/titlepage page, then any manifest image. Both feed the
  existing FFmpeg WebP pipeline, so they match every other thumbnail in the vault
- `.pdf` and `.epub` now resolve to `application/pdf` / `application/epub+zip`, so the metadata panel and the
  `/api/original/{id}` route report the real type instead of `application/octet-stream`
- A document with no cover to show — an EPUB without images, a PDF that is encrypted or damaged — gets a drawn
  fallback card instead of a blank cell: format badge, title and author, set in the system font (Segoe UI on
  Windows, so Cyrillic and other scripts render), encoded through the same FFmpeg WebP pipeline
- The metadata panel now reads what the container says: page count for PDFs, title and author from an EPUB's
  package metadata
- PDFs open inline: the fullscreen modal embeds an `<iframe>` served as
  `application/pdf`, which WebView2 renders with its built-in viewer. Double-click and Enter in the gallery open a
  PDF there rather than the system's PDF reader — the "which formats does the app display itself" list used to live
  in three files and had not heard about PDFs; it is now one list in `frontend/src/utils/extensions.ts`
- The right-panel preview does not load a PDF by itself: it keeps the thumbnail and shows a **Load PDF** button,
  so moving quickly through a folder never starts the viewer for files that were only glanced at. Double-click
  and Enter open the document straight away in the modal

### Changes

- One module answers "what kind of file is this?": `frontend/src/utils/extensions.ts` holds the image, video and
  PDF sets plus `extensionOf` / `formatLabel`, replacing the copies that the panel, the modal, the gallery cells,
  the list rows, the Enter shortcut and drag-and-drop each kept for themselves
- Building TagLoom now requires Go 1.26 (pulled in by go-pdfium), and the executable is about 11 MB larger
  (29.9 MB vs 18.3 MB) because the 5.7 MB PDFium WASM module and wazero are embedded — no runtime dependency
  beyond FFmpeg

### Fixes

- Dropping a `.pdf` or `.epub` onto the window now offers the import menu: `SUPPORTED_EXTENSIONS` on the frontend
  mirrors `utils.SupportedExtensions` in Go by hand, and the two new formats had not been added to it

- Search no longer stops at 500 files: the query is now a filter in `GetFiles`, so the gallery counts every
  match and lazy-loads page by page as you scroll, the same way folder and tag filters do
- Search now combines with the other filters (folder, tags, formats, rating, favorites) instead of replacing
  them, and search results are sorted by the current sort order and carry their tags
- Batch edits scoped to the gallery respect the active search

## [0.6.0] - 26-09-2026

### New Features

- Tag Manager: one window to browse, create, edit, re-parent, reorder, merge and delete tags — virtual
  list for large tag sets, search by name or alias, multi-select, keyboard navigation, and drag to re-parent
- Escape steps back through the layers: selection, then filter text, then the window
- Merge tags with a preview of what moves — file links, aliases, child tags — counted before anything runs;
  a merged-away tag's name becomes an alias of the target, so saved searches keep matching
- Deleting tags now reports how many files are affected and, when a tag has children, asks whether to move
  them up a level, promote them to top level, delete them too, or refuse
- Categories are top-level grouping tags: they cannot be nested or assigned to files, and a vault migration
  flattens any that were
- Tag Manager filter and sort menus — show only unused tags, categories, uncolored tags or top-level ones,
  and order each level by name, file count or creation date; the footer reads `filtered 12 / 84 tags`
- Ctrl+F focuses the Tag Manager's filter box
- Renaming a tag onto a name that already exists offers to merge it into that tag instead
- Deleting a tag that is on files offers to merge it instead, which keeps every file link
- The details pane reports how a tag is used (files, child tags, aliases, files under it) and, when a tag
  that is already on files is turned into a category, links straight to those files

### Changes

- Tag aliases are passed as a list instead of a comma-separated string; a comma in a name or alias no longer
  splits it
- Tag rows carrying no files are dimmed so the used vocabulary stands out, and each row shows edit,
  move up, move down and delete on hover
- Checking "Is Category" clears the parent picker, so the form cannot promise a parent the save would drop
- The right panel's tag list refreshes after a merge or a delete, so it never lists a tag that is gone
- QuickActions: icons 15 -> 20 and buttons 28px -> 32px, App Settings move below Tag Manager

### Fixes

- Deleting a tag from the tag editor no longer leaves child tags pointing at a removed row
- A tag could be given a parent inside itself, hiding its whole subtree; the backend now refuses it
- An in-app drag (re-parenting a tag) no longer flashes the "Drop files to import" overlay
- App shortcuts (Ctrl+A, Ctrl+B, Ctrl+C, Ctrl+R) no longer fire while the Tag Manager is open — Ctrl+A
  selects tags there instead of every file in the gallery
- Clearing a tag's parent in the form really moves it to the top level now; the field used to send
  "leave it alone", so the tag quietly stayed where it was
- Escape while a merge or delete dialog is open closes the dialog only — it no longer also clears the
  selection behind it
- The Tag Manager's details pane no longer opens empty: reading a tag's aliases could throw before the form
  was filled, leaving the name blank and Save disabled for every change except typing a name
- Selecting a tag always leaves the "New tag" pane — an empty create form no longer sits on top of a
  selected tag, which also left Save with nothing to save

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
