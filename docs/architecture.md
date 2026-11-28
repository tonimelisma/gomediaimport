# Architecture

gomediaimport is a CLI that imports and organizes photos and videos from any source directory or
mounted removable volume into a destination directory. All Go code is one `main` package in
`cmd/gomediaimport/`.

## Components

- `cmd/gomediaimport/main.go`: the entry point. `run(osArgs []string)` (extracted from `main()`
  so tests pass their own arguments; no global state) parses flags with go-arg
  (`arg.NewParser` + `Parse`), loads the YAML config strictly (`KnownFields` rejects unknown
  keys), validates it, and dispatches to a one-off or `source_directory` import, a configured
  removable-volume import, or a `volumes` subcommand. Precedence is flags, then the config file
  (`os.UserConfigDir()/gomediaimport/config.yaml`), then defaults; `wasFlagProvided(osArgs, flag)`
  is a pure function that lets an explicit `--flag=false` beat the file
  ([ADR 0003](adr/0003-strict-config-with-explicit-overrides.md)). `--help` appends the resolved
  config path. `checksum_duplicates` defaults to true (`--no-checksum-duplicates` opts out);
  `--quiet` forces `Verbose=false`.
- `cmd/gomediaimport/version.go`: `resolveVersion` returns the version the release stamps with
  `-ldflags -X main.version`, or the module version from the build info.
- `cmd/gomediaimport/import.go`: the orchestrator. `FileStatus` and `FileInfo` (source and
  destination paths, checksum, creation time, optional `VideoMetadata`, media category, status)
  are the central types. `importMedia()` runs enumeration, `planDestinations()`, `copyFiles()`,
  `deleteOriginalFiles()`, `cleanupSourceArtifacts()`, `printSummary()` and
  `ejectAfterImport()`; an error in planning, copying, deletion or cleanup stops every later
  destructive phase and the eject ([ADR 0004](adr/0004-errors-stop-destructive-phases.md)).
  `planDestinations()` plans in two passes (non-sidecars, then sidecars); date-time renames first
  sort by capture time and natural source-filename order, so same-second suffixes follow the
  camera's sequence. `progressTracker` owns the ANSI progress line with atomic size tracking.
- `cmd/gomediaimport/file_operations.go`: enumeration with `filepath.WalkDir`, which separates
  importable files from typed cleanup targets: top-level system trash, exact Sony
  `[PRIVATE/]M4ROOT/THMBNL` directories, Sony `CLIP/*.XML` and AppleDouble `._*` files never
  reach media classification ([ADR 0005](adr/0005-source-artifacts-never-import.md)). Also
  duplicate detection (size+timestamp index, xxHash64), unique filenames, stream copies with a
  written-byte check, and disk eject.
- `cmd/gomediaimport/removable_volumes.go` and `removable_volumes_{darwin,linux}.go`: the
  label-based volume workflow ([ADR 0002](adr/0002-label-only-removable-volume-selectors.md)).
  `volumes list` discovers mounted removable volumes, `volumes add` saves a mounted label in
  `removable_volumes`, and a plain `gomediaimport` imports every mounted volume whose label is
  saved. Discovery is no-cgo: macOS uses `Getfsstat` plus `getattrlist`, accepting
  `MNT_REMOVABLE` volumes and local `/Volumes/...` device mounts; Linux uses
  `/proc/self/mountinfo`, `/sys/dev/block` and `/dev/disk/by-label`.
- `cmd/gomediaimport/metadata.go`: image EXIF/XMP through `bep/imagemeta` (callback API,
  `ShouldHandleTag` accepts all tags); `resolveImageFormat()` maps file types to imagemeta
  formats. `extractMetadata(fileInfo)` returns a `mediaMetadata` wrapper, so MP4/MOV-family
  videos also carry `VideoMetadata` decoded by `github.com/tonimelisma/videometa`
  (`QUICKTIME | CONFIG | MAKERNOTES | XML`): the chosen timestamp, its provenance or fallback
  reason, codec and config, GPS, camera make and model, and non-fatal warnings. Unsupported or
  undecodable containers fall back to the file's mtime without stopping enumeration.
- `cmd/gomediaimport/media_types.go`: `MediaCategory` (ProcessedPicture, RawPicture, Video,
  RawVideo, Sidecar), `FileType` constants, the extension table `fileTypes`, and `SidecarAction`
  with per-extension defaults and overrides.

## Data

There is no state file or database: each run derives everything from the source and
destination filesystems ([ADR 0001](adr/0001-idempotent-imports-without-state.md)). The config
file is the only persisted input, written only by `volumes add`.

## Flows

1. `run()` resolves the config (defaults, YAML, then flags via `wasFlagProvided`) and dispatches
   to a `volumes` subcommand, a configured removable-volume import, or a one-off or
   `source_directory` import.
2. `enumerateFiles()` walks the source, skips symlinks, prunes system trash and the exact Sony
   support paths into cleanup targets, keeps files with media extensions and extracts their
   timestamps.
3. Planning assigns `YYYY/MM` subdirectories and/or `YYYYMMDD_HHMMSS.ext` names.
4. Duplicates are found by an O(1) `fileSizeTime` lookup and, by default, confirmed by xxHash64
   for existing destination files and earlier files in the batch.
5. `copyFiles()` runs a worker pool (default 4) of stream copies with a written-byte check and an
   explicit `Close()` error check, removes partial files on failure, and tracks progress
   ([ADR 0006](adr/0006-concurrent-copy-worker-pool.md)).
6. Optionally, originals are deleted, then the recorded trash, Sony and AppleDouble artifacts;
   any failure stops ejection.
7. Errors go to stderr and the exit code is 1.

## Failure handling

`copyFiles()`, `deleteOriginalFiles()` and `cleanupSourceArtifacts()` accumulate errors and
return them all through `errors.Join()`. Checksum and `setFileTimes` failures are warnings on
stderr, never swallowed. A failure in one configured removable volume does not stop the others,
but the run still exits non-zero.

## Releases

`.github/workflows/release.yml`, triggered by CI succeeding on main, builds darwin and linux
binaries for amd64 and arm64, writes `SHA256SUMS`, attests them and publishes the release;
`script/release-version` chooses the version from `VERSION` and the tags
([ADR 0007](adr/0007-attested-release-from-ci-green-main.md)).
