# Changelog

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
