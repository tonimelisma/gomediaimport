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
8. `git push` after all the above
9. Create a GitHub release (no binaries) — tag with semantic version, include changelog entry in the release body. Use `gh release create` or the GitHub MCP tool. **This must be the very last step** — no commits after the tag. If you need to fix something after tagging, delete the release/tag and re-create it.

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

- **main.go** — Entry point, `run(osArgs []string)` function (extracted from `main()` for testability, takes os args as parameter — no global state), CLI parsing via `arg.NewParser`+`Parse` (`go-arg`), YAML config loading, config validation, `wasFlagProvided(osArgs, flag)` pure function for proper boolean override semantics. Three-tier config precedence: CLI flags > YAML file (`os.UserConfigDir()/gomediaimport/config.yaml`) > hardcoded defaults. `watch` subcommand dispatches to `runWatch(cfg, *watchArgs)`. `WatchConfig` sub-struct embedded in `config` via `yaml:",inline"` for flat YAML keys. `--quiet` flag suppresses all non-error stdout output (`cfg.Quiet` forces `cfg.Verbose=false`).
- **import.go** — Core orchestrator. Defines `FileStatus` type with typed constants and `FileInfo` struct (central data type tracking source/dest paths, checksums, creation time, media category, status). `importMedia()` coordinates enumeration → `planDestinations()` → `copyFiles()` → `deleteOriginalFiles()` → `printSummary()` → `ejectAfterImport()`. `printConfig()` extracted for verbose output. `planDestinations()` handles two-pass destination planning (non-sidecars then sidecars). `progressTracker` type encapsulates ANSI progress display with atomic size tracking. `ejectAfterImport(sourceDir, quiet)` returns error, gates output on `quiet` flag. `deleteOriginalFiles()` accumulates and returns errors (not silently swallowed). `copyFiles()` returns all errors via `errors.Join()`.
- **file_operations.go** — File enumeration via `filepath.WalkDir` (with symlink skipping), duplicate detection (size+timestamp index or xxHash64 checksum), `exists()` returns `(bool, error)`, unique filename generation (appends `_001` through `_999999` on conflicts), `copyFile()` stream copy with `Sync()` + explicit `Close()`, cross-platform disk eject (`ejectDrive` dispatches to `ejectDriveDarwin` on macOS or `ejectDriveLinux` on Linux).
- **metadata.go** — EXIF/XMP extraction using `bep/imagemeta` (callback-based API with `ShouldHandleTag` set to accept all tags). `resolveImageFormat()` maps FileType/extension to imagemeta format constants. Falls back to filesystem mtime for unsupported formats. Video metadata via MP4/MOV mvhd box parsing.
- **media_types.go** — `MediaCategory` (ProcessedPicture, RawPicture, Video, RawVideo, Sidecar) and `FileType` constants. Extension-to-type mapping in `fileTypes` slice. `SidecarAction` type with per-extension defaults and overrides.
- **watch.go** — Watch mode orchestrator. `runWatch(cfg, *watchArgs)` resolves `plistPath()` and passes it to install/uninstall/status functions (injectable for testing). `installLaunchAgent(cfg, pPath)` generates plist via `howett.net/plist`, writes to `~/Library/LaunchAgents/`, bootstraps with `launchctl`. `runWatchImport(cfg, volumesDir, diskutilFn)` scans configurable volumes dir, filters via `filterVolume()`, calls `importMedia()` for each match, collects all errors via `errors.Join()`. Plays completion sound via `playSound()` using `afplay` (configurable via `watch_sound`, default `Hero`). Verbose logging at each filter stage. `watchStatus(cfg, pPath)` warns if binary path in plist doesn't exist.
- **diskutil.go** — `VolumeInfo` struct, `diskutilInfoFn` type, `diskutilInfoReal()` implementation, `filterVolume(mountPoint, cfg, diskutilFn)` multi-stage pipeline (ejectable check, DCIM folder, glob allowlist) with verbose logging at each rejection stage — takes function parameter for testability. `parseDiskutilPlist()` for parsing raw plist data.
### Program Flow

1. `run()` → parse config (defaults → YAML → CLI overrides via `wasFlagProvided`). If `watch` subcommand, dispatch to `runWatch()`.
2. `enumerateFiles()` — WalkDir source dir, skip symlinks, filter by media extensions, extract EXIF dates
3. Plan destinations — date-based subdirs (`YYYY/MM`) and/or timestamp-based filenames (`YYYYMMDD_HHMMSS.ext`)
4. Detect duplicates — O(1) lookup via `fileSizeTime` index, compare against existing destination files and within current batch
5. `copyFiles()` — concurrent worker pool (default 4 workers), stream copy with `Sync()` + explicit `Close()` error check, cleanup partial files on failure, progress tracking with ANSI sticky line
6. Optional: delete originals, eject drive (macOS via `diskutil eject`, Linux via `udisksctl`/`umount`)
7. Errors go to stderr, exit code 1 on failure

### Watch Mode Flow

1. `launchd` triggers binary via `StartOnMount` → `gomediaimport watch --run`
2. Load config from `os.UserConfigDir()/gomediaimport/config.yaml`
3. Scan `/Volumes`, filter each through `filterVolume()` (diskutil → DCIM → allowlist)
4. For each passing volume, call `importMedia()` with source set to mount point
5. Exit 0 if all succeed, exit 1 if any failed

### Design Decisions

- **Idempotent**: safe to re-run; duplicates detected by size+checksum, never re-copied
- **No state file/database**: all state derived from filesystem inspection each run
- **No global state**: `run(osArgs)` takes args as parameter; `wasFlagProvided(osArgs, flag)` is a pure function; `filterVolume` and `runWatchImport` take injectable function parameters. Tests need no save/restore of globals.
- **Boolean CLI override**: `wasFlagProvided()` checks the passed `osArgs` so CLI `--flag=false` correctly overrides a `true` config file value
- **Concurrent copying**: worker pool with configurable `--workers` (default 4); enumeration remains sequential; `progressTracker` type manages progress display
- **Warnings to stderr**: checksum errors and setFileTimes failures logged to stderr, not swallowed
- **xxHash64 for checksums**: faster than CRC32, 64-bit collision space is more than sufficient for non-adversarial duplicate detection
- **All errors propagated**: `copyFiles()`, `deleteOriginalFiles()`, and `runWatchImport()` accumulate errors and return all of them via `errors.Join()`, ensuring the tool exits non-zero when operations fail
- **Watch mode is one-shot**: LaunchAgent triggers the binary on every mount; the binary scans all volumes, imports, and exits. No daemon, no polling.
- **Dependency injection for testing**: `filterVolume` and `runWatchImport` take a `diskutilInfoFn` parameter; `runWatchImport` takes a configurable `volumesDir` path; `installLaunchAgent`, `uninstallLaunchAgent`, `watchStatus` take a `pPath` parameter. Tests pass mock functions and temp paths directly — no package-level globals needed.
- **Quiet mode**: `--quiet` / `-q` suppresses all non-error stdout output; forces `Verbose=false`. Stderr warnings/errors always print. Interactive commands (`watch --install/--uninstall/--status`) are NOT suppressed.
