# Changelog

## [v1.5.1] - 2026-03-17

### Removed
- **macOS notifications**: removed `sendNotification()` and `watch_notifications` config option. The `osascript`-based notification feature silently fails on macOS Sequoia due to per-app notification permission changes, and all available alternatives have significant drawbacks.

## [v1.5.0] - 2026-03-17

### Features
- **`--quiet` / `-q` flag**: suppresses all non-error stdout output. Stderr warnings/errors always print. Interactive commands (`watch --install/--uninstall/--status`) are NOT suppressed. Forces `Verbose=false`.
- **Watch mode verbose logging**: when `verbose: true`, `filterVolume()` logs each filter stage (ejectable, DCIM, allowlist) with accept/reject reason. `runWatchImport()` logs config summary and volume scan progress. Output goes to `~/Library/Logs/gomediaimport.out.log` when running via LaunchAgent.

### Bug Fixes
- **`runWatchImport()` now collects all errors**: previously only kept the first error via `firstErr`. Now accumulates all errors and returns them via `errors.Join()`.
- **`ejectDriveMacOS()` no longer prints to stdout**: removed unconditional `fmt.Printf` calls. All messaging moved to `ejectAfterImport()`, gated by `quiet` flag.
- **`ejectAfterImport()` now returns error**: previously returned void, swallowing eject failures.
- **Permission-denied warning uses stderr**: `enumerateFiles` WalkDir callback now writes "Permission denied" warning to stderr instead of stdout.

### Improvements
- **Reduced mutex scope in `copyFiles` worker loop**: stderr writes and `setFileTimes` warnings no longer hold the mutex.
- **Extracted `printConfig()`**: 10 inline `fmt.Println` calls in `importMedia()` moved to a named function.
- **Moved `copyFile()` to `file_operations.go`**: pure function relocated to where other file operations live.
- **`plistPath` injectable for testing**: `installLaunchAgent`, `uninstallLaunchAgent`, and `watchStatus` now take a `pPath` parameter. All three are testable with temp paths.

### Testing
- Added `TestRunWatchImportCollectsAllErrors` — verifies all volume errors are returned, not just the first.
- Added `TestWatchStatusBinaryMissingWarning` — verifies "WARNING: binary not found" output when plist references a missing binary.
- Removed `t.Skip` from `TestInstallRefusesIfAlreadyInstalled` and `TestUninstallWhenNotInstalled` — now deterministic with temp paths.
- Added `TestRunQuietSuppressesOutput` — verifies `--quiet` produces no stdout.
- Added `TestFilterVolumeVerboseLogging` — verifies verbose filter rejection output.
- Added `TestRunWatchImportVerboseLogging` — verifies verbose scan/config logging.
- Added `TestPlanDestinations` — direct unit tests for organize-by-date, rename-by-datetime, plain copy, sidecar-follows-parent, orphaned sidecar, and sidecar-delete.
- Added `TestProgressTracker` — tests non-TTY output (no ANSI codes), verbose=false (no output), and finish() in non-TTY mode.

## [v1.4.0] - 2026-03-17

### Bug Fixes
- **`deleteOriginalFiles()` now returns errors**: previously silently swallowed delete failures and always returned nil. Now accumulates errors and returns them via `errors.Join()`, ensuring exit code 1 when files fail to delete.
- **`copyFiles()` returns all errors**: previously only returned the first error, discarding subsequent failures. Now returns all errors via `errors.Join()`.
- **`sendNotification()` no longer leaks zombie processes**: added goroutine to reap osascript processes after `cmd.Start()`.
- **`humanReadableDuration()` handles zero/negative durations**: guard at top returns "0s" for non-positive durations.

### Improvements
- **`watch --status` warns about stale binary path**: reads the plist and warns if the binary referenced in `ProgramArguments[0]` no longer exists on disk.
- **Eliminated global `args` variable**: `run()` now takes `osArgs []string` parameter; all tests simplified (no save/restore of globals).
- **`filterVolume()` takes `diskutilInfoFn` parameter**: replaced package-level `diskutilInfoFunc` global with explicit function parameter for testability.
- **`runWatchImport()` takes injectable `volumesDir` and `diskutilFn`**: fully testable without mocking `/Volumes`. New `TestRunWatchImportScansVolumes` exercises the full scan-filter-import loop.
- **`WatchConfig` sub-struct**: watch-related config fields grouped into embedded struct with `yaml:",inline"` (no config file format change).
- **Extracted `planDestinations()`**: two-pass destination planning pulled out of `importMedia()` into a named function.
- **Extracted `progressTracker` type**: ANSI progress display encapsulated in a dedicated type with atomic size tracking, separated from copy logic.
- **Extracted `printSummary()` and `ejectAfterImport()`**: further decomposition of `importMedia()` for readability.

### Testing
- All `run()` tests now pass `--config` pointing to a temp file, isolating from host `~/.gomediaimportrc`.
- Added `TestDeleteOriginalFilesReturnsError` — verifies error return on delete failure.
- Added `TestHumanReadableDurationEdgeCases` — verifies zero, negative, and sub-second durations.
- Added `TestRunWatchImportScansVolumes` — end-to-end test of the watch import scan loop with mock volumes.
- Enhanced `TestWatchSubcommandParsing` — actually parses CLI args via `arg.NewParser`.
- Deleted empty `TestWatchRunPrintsTimestamp` (now covered by `TestRunWatchImportScansVolumes`).
- Conditional tests (`TestInstallRefusesIfAlreadyInstalled`, `TestUninstallWhenNotInstalled`) now use `t.Skip` with clear messages.

## [v1.3.0] - 2026-03-17

### Breaking Changes
- **`SourceDir` is now `--source` flag**: the source directory is no longer a positional argument. Use `--source /path/to/source` instead. This change was required to support subcommands with go-arg.

### Features
- **Auto-import watch mode** (`watch` subcommand, macOS only): install a LaunchAgent that automatically imports media when SD cards or camera cards are mounted. Uses `StartOnMount` to trigger on any filesystem mount, then filters volumes by diskutil properties, DCIM folder presence, and optional volume name allowlist.
  - `gomediaimport watch --install` — install the LaunchAgent
  - `gomediaimport watch --uninstall` — remove the LaunchAgent
  - `gomediaimport watch --status` — show install status and watch configuration
- **macOS notifications**: optional `display notification` alerts on card detection, import completion, and errors (configurable via `watch_notifications`)
- **Volume filtering pipeline**: multi-stage filter (ejectable check, DCIM folder, glob-pattern allowlist) prevents importing from non-camera volumes

### Configuration
- New optional config keys: `watch_require_dcim` (default: true), `watch_volumes` (default: all), `watch_notifications` (default: true)
- All watch settings are top-level in `~/.gomediaimportrc`

### New Dependency
- Added `howett.net/plist` for LaunchAgent plist generation and `diskutil info -plist` parsing

## [v1.2.0] - 2026-03-01

### Breaking Changes
- **Go 1.25 required**: minimum Go version bumped from 1.24 to 1.25
- **CR3 format no longer supported for EXIF extraction**: Canon CR3 raw files fall back to filesystem mtime for date extraction (CR3 was never widely tested)

### Improvements
- **Migrate EXIF library from evanoberholster/imagemeta to bep/imagemeta**: the new library is actively maintained (by Hugo's lead developer), supports more formats, and uses a callback-based API with proper EXIF IFD traversal
- **Dependency cleanup**: removed 6 transitive dependencies (zerolog, msgp, pkg/errors, philhofer/fwd, mattn/go-colorable, tinylib/msgp); added only golang.org/x/text

### CI
- Bumped Go version in CI from 1.24 to 1.25
- Bumped staticcheck from v0.6.0 to v0.7.0

## [v1.1.3] - 2026-02-28

### Bug Fixes
- **Fix video filename timezone mismatch**: video files (MP4, MOV, etc.) got UTC-based filenames while images got local-time filenames from EXIF, causing videos to be off by the local timezone offset. For recordings near midnight, this resulted in videos being dated on the wrong day. Removed erroneous `.UTC()` call so video times use local time, matching EXIF image behavior.

## [v1.1.2] - 2026-02-09

### Features
- Added `--version` flag to print version and exit
- Version can be set at build time via `-ldflags "-X main.version=..."`, defaults to `dev`

## [v1.1.1] - 2026-02-09

### Bug Fixes
- **Fix copy error not reported**: `copyFiles()` now accumulates errors when `copyFile()` fails, ensuring non-zero exit code on file copy failures (data integrity bug)

### Testing
- Added `TestCopyFilesAccumulatesCopyError` — verifies `copyFiles()` returns error when a source file is missing
- Added `TestSetFinalDestinationFilenameMultipleCollisions` — verifies `_001`, `_002` suffixes on filename collisions
- Added `TestSetFinalDestinationFilenameNoRename` — verifies original filename preserved when `RenameByDateTime=false`
- Added `TestSetFinalDestinationFilenameDuplicateInBatch` — verifies in-batch duplicate detection marks `StatusPreExisting`
- Added `TestRunBooleanOverrideFalse` — verifies `--verbose=false` CLI overrides `verbose: true` in config
- Added `TestRunCustomConfigPath` — verifies `--config` flag uses a custom YAML file
- Added `TestRunWorkersOverride` — verifies `--workers` CLI flag overrides config file value

### Documentation
- Rewrote README.md: platform-neutral language, added missing CLI flags (`--sidecar-default`, `--workers`), added sidecar file types section, updated examples and How It Works
- Updated CLAUDE.md: added docs update and GitHub release creation to DOD, updated architecture section
- Updated example config file (`gomediaimportrc`): added `sidecar_default`, `sidecars`, and `workers` options
- Removed `ROADMAP.md`

### Maintenance
- Fixed `go fmt` drift in `import_test.go` and `metadata_test.go`
- Upgraded local `golangci-lint` to v2

## [v1.1.0] - 2025-02-08

### Features
- **Multithreaded copying**: concurrent worker pool with `--workers` flag (default 4), size-interleaved job scheduling for balanced load
- **Sidecar file handling**: configurable per-extension actions (copy/delete/ignore) with `sidecar_default` and `sidecars` config keys
- **Video metadata extraction**: EXIF-based creation date for MP4, MOV, M4V, 3GP, and 3G2 files
- **Optional source directory**: source dir can be omitted on CLI when set in YAML config file
- **TTY-aware progress output**: ANSI sticky progress line in terminals, plain output when piped

### Bug Fixes
- **Fix `isDuplicate()` crash**: `os.Stat` errors (e.g. permission denied) caused a nil pointer panic; signature changed to `(bool, error)` with proper error propagation
- **Fix progress bar interleaving**: file copy output no longer interleaves with the progress bar in terminal mode
- **Fix boolean config override**: `--flag=false` on CLI now correctly overrides `true` in config file via `wasFlagProvided()` helper
- **Fix `copyFile` reliability**: explicit `Sync()` + `Close()` error check, partial file cleanup on failure
- **Fix `exists()` error handling**: non-`IsNotExist` errors are now returned instead of silently treated as "not exists"
- **Fix division-by-zero**: guard against zero total size in progress calculation
- **Fix exit codes**: non-zero exit on failure
- **Log warnings to stderr**: checksum and setFileTimes warnings go to stderr, not stdout
- **Validate destination directory**: check that parent of dest dir exists before starting

### Performance
- **xxHash64 checksums**: replaced CRC32 with xxHash64 for faster duplicate detection with better collision resistance
- **O(1) duplicate detection**: `fileSizeTime` index replaces O(n²) scan of previous files

### Refactoring
- Upgraded yaml.v2 to yaml.v3
- Introduced typed `FileStatus` constants
- Use `filepath.WalkDir` (not `Walk`) with symlink skipping
- Eject drive only on successful import, not via defer
- Removed deprecated `ioutil` usage

### Testing
- Added tests for `isNameTakenByPreviousFile` (table-driven)
- Added symlink skipping test for `enumerateFiles`
- Added zero-byte file enumeration and copy test
- Expanded `isDuplicate` tests including stat error propagation
- Added sidecar enumeration tests
- Added actual file copy verification test

### CI / DevOps
- Added `go fmt` check, coverage reporting, and `golangci-lint` to CI
- Set up Dependabot for Go modules and GitHub Actions (weekly)
- Added govulncheck and staticcheck to CI pipeline
- Added CI workflow for Go 1.23 and 1.24

## [v1.0.0] - 2024-01-01

Initial release.
