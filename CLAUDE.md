# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Documentation

- [README.md](README.md) — Project overview and usage
- [CHANGELOG.md](CHANGELOG.md) — Release history

## Ownership & Responsibility

You are the sole developer of this codebase. That means:

- **All tests must pass.** If any test is failing, that's your bug — fix it before doing anything else.
- **All code must compile.** Never leave the repo in a broken state.
- **No uncommitted work.** Before finishing a session, everything should be committed with a clear message. Don't leave files modified and unstaged.
- **No untracked generated files.** Build artifacts belong in `.gitignore`, not lingering in `git status`.
- **You own the CI too.** If CI is red, fix it. Don't ignore it.
- **`go vet` must be clean.** No warnings, no excuses.

## Definition of Done (DOD)

Before considering any task complete, you must:

1. `go build ./cmd/gomediaimport` — compiles without errors
2. `go test -race ./cmd/gomediaimport` — all tests pass, no race conditions
3. `go vet ./cmd/gomediaimport` — no warnings
4. If on Toni's MacBook Pro (`Tonis-MacBook-Pro.local`), `go install ./cmd/gomediaimport` — installs to `~/go/bin/`
5. Update documentation — ensure `CLAUDE.md`, `README.md`, `CHANGELOG.md`, and any other docs reflect the changes made
6. `git status` — clean working tree (no modified, staged, or untracked files that shouldn't be there)
7. Commits have clear, conventional-style messages (e.g. `fix:`, `feat:`, `refactor:`, `test:`, `docs:`)
8. If asked to push, `git push` after all the above
9. Create a GitHub release (no binaries) — tag with semantic version, include changelog entry in the release body. Use `gh release create` or the GitHub MCP tool.

## Build & Test Commands

```bash
# Build
go build ./cmd/gomediaimport

# Install globally (with version embedded)
go install -ldflags "-X main.version=$(git describe --tags --always)" ./cmd/gomediaimport

# Run all tests
go test ./cmd/gomediaimport

# Run all tests with race detector
go test -race ./cmd/gomediaimport

# Run a single test
go test -run TestEnumerateFiles ./cmd/gomediaimport

# Verbose test output with coverage
go test -v -cover ./cmd/gomediaimport

# Format and vet
go fmt ./...
go vet ./...
```

## Architecture

gomediaimport is a CLI tool that imports and organizes media files (photos/videos) from any source directory into a destination directory. All Go code lives in a single package under `cmd/gomediaimport/`.

### Key Files

- **main.go** — Entry point, `run()` function (extracted from `main()` for testability), CLI parsing (`go-arg`), YAML config loading, config validation, `wasFlagProvided()` helper for proper boolean override semantics. Three-tier config precedence: CLI flags > YAML file (`~/.gomediaimportrc`) > hardcoded defaults.
- **import.go** — Core orchestrator. Defines `FileStatus` type with typed constants and `FileInfo` struct (central data type tracking source/dest paths, checksums, creation time, media category, status). `importMedia()` coordinates enumeration → copying → deletion → eject. Builds `fileSizeTime` index for O(1) duplicate lookup. `copyFiles()` uses a worker pool (`--workers` flag, default 4) with size-interleaved job scheduling for balanced load.
- **file_operations.go** — File enumeration via `filepath.WalkDir` (with symlink skipping), duplicate detection (size+timestamp index or xxHash64 checksum), `exists()` returns `(bool, error)`, unique filename generation (appends `_001` through `_999999` on conflicts), macOS disk eject.
- **metadata.go** — EXIF extraction using `imagemeta` library. Falls back to filesystem mtime. Video metadata via MP4/MOV mvhd box parsing.
- **media_types.go** — `MediaCategory` (ProcessedPicture, RawPicture, Video, RawVideo, Sidecar) and `FileType` constants. Extension-to-type mapping in `fileTypes` slice. `SidecarAction` type with per-extension defaults and overrides.

### Program Flow

1. `run()` → parse config (defaults → YAML → CLI overrides via `wasFlagProvided`)
2. `enumerateFiles()` — WalkDir source dir, skip symlinks, filter by media extensions, extract EXIF dates
3. Plan destinations — date-based subdirs (`YYYY/MM`) and/or timestamp-based filenames (`YYYYMMDD_HHMMSS.ext`)
4. Detect duplicates — O(1) lookup via `fileSizeTime` index, compare against existing destination files and within current batch
5. `copyFiles()` — concurrent worker pool (default 4 workers), stream copy with `Sync()` + explicit `Close()` error check, cleanup partial files on failure, progress tracking with ANSI sticky line
6. Optional: delete originals, eject drive (macOS only via `diskutil eject`)
7. Errors go to stderr, exit code 1 on failure

### Design Decisions

- **Idempotent**: safe to re-run; duplicates detected by size+checksum, never re-copied
- **No state file/database**: all state derived from filesystem inspection each run
- **Boolean CLI override**: `wasFlagProvided()` checks `os.Args` so CLI `--flag=false` correctly overrides a `true` config file value
- **Concurrent copying**: worker pool with configurable `--workers` (default 4); enumeration remains sequential
- **Warnings to stderr**: checksum errors and setFileTimes failures logged to stderr, not swallowed
- **xxHash64 for checksums**: faster than CRC32, 64-bit collision space is more than sufficient for non-adversarial duplicate detection
- **Copy errors are fatal**: `copyFiles()` accumulates errors and returns the first one, ensuring the tool exits non-zero when files fail to copy
