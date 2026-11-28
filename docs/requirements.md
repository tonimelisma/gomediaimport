# Requirements

Current behavior of gomediaimport. Each requirement is one EARS sentence ending `[verified]` (a
test tagged `Validates: <ID>` proves it) or `unverified: <why>`. IDs are never reused or
renumbered. Future work lives in GitHub issues.

## Configuration

- **GMI-001** The importer shall resolve every setting from the built-in defaults, then the YAML
  config file, then the command-line flags, each later source overriding the earlier.
  [verified]
- **GMI-002** When a boolean flag is given explicitly as `--flag=false`, the importer shall use
  false even if the config file sets it true. [verified]
- **GMI-003** If the config file contains an unknown, misspelled or removed key, at the top level
  or under a removable volume, then the importer shall fail with a config error instead of
  ignoring it. [verified]
- **GMI-004** The importer shall read its config from `os.UserConfigDir()/gomediaimport/config.yaml`
  unless `--config` names another file, and `--help` shall show the resolved default path.
  [verified]
- **GMI-005** If the configuration has a negative worker count, an invalid sidecar action or an
  unusable destination, then the importer shall fail before touching any file. [verified]
- **GMI-006** If no source is given by `--source`, `source_directory` or a configured mounted
  removable volume, or the given source is not a directory, then the importer shall exit with an
  error. [verified]
- **GMI-007** When `--source` is omitted and no removable volume labels are configured, the
  importer shall import from the config file's `source_directory`. [verified]
- **GMI-008** While `--quiet` is set, the importer shall print nothing to stdout except errors and
  shall force verbose output off; warnings and errors still go to stderr. [verified]
- **GMI-009** When run with `--version`, the importer shall print `gomediaimport <version>`, where
  the version is the one stamped at build time or, unstamped, the module version Go recorded.
  [verified]
- **GMI-010** The importer shall marshal and unmarshal its config without losing any field.
  [verified]

## Enumeration and metadata

- **GMI-020** The importer shall walk the source directory recursively and classify each file by
  extension as a processed picture, raw picture, video, raw video or sidecar, ignoring every
  other file. [verified]
- **GMI-021** The importer shall skip symbolic links during enumeration. [verified]
- **GMI-022** The importer shall exclude top-level system trash (`.Trashes`, `.Trash`,
  `.Trash-UID`, `$RECYCLE.BIN`), Sony XAVC `[PRIVATE/]M4ROOT/THMBNL` directories,
  `[PRIVATE/]M4ROOT/CLIP/*.XML` companions and AppleDouble `._*` files before classification, and
  record them as cleanup targets. [verified]
- **GMI-023** The importer shall take a picture's creation time from its EXIF/XMP metadata and
  fall back to the file's modification time.
  unverified: no committed image fixture carries EXIF dates; only the video path has fixtures.
- **GMI-024** For `.mp4`, `.mov`, `.m4v`, `.3gp` and `.3g2` files, the importer shall take the
  creation time from QuickTime metadata through videometa, preferring QuickTime-native dates over
  vendor `DateTimeOriginal`, and keep timezone, GPS, camera make and model. [verified]
- **GMI-025** If a video container is unsupported or its metadata cannot be decoded, then the
  importer shall use the file's modification time and continue enumerating. [verified]
- **GMI-026** The importer shall import zero-byte media files like any other. [verified]

## Destination planning

- **GMI-030** Where `organize_by_date` is set, the importer shall place each file under
  `YYYY/MM` of its creation time. [verified]
- **GMI-031** Where `rename_by_date_time` is set, the importer shall name each file
  `YYYYMMDD_HHMMSS.ext` and give same-second files `_001`, `_002`, ... suffixes in capture-time
  and natural source-filename order. [verified]
- **GMI-032** Without renaming, the importer shall keep the source filename and choose a unique
  name when another file, existing or planned in the same batch, already takes it. [verified]
- **GMI-033** The importer shall treat a file as a duplicate, and not copy it, when an existing
  destination file or an earlier file in the batch has the same size and timestamp and, unless
  checksum verification is disabled, the same xxHash64 checksum. [verified]
- **GMI-034** The importer shall copy `xmp` and `srt` sidecars and delete `thm`, `ctg`, `aae`,
  `lrf`, `mpl` and `cpi` sidecars by default, apply `sidecar_default` and per-extension
  overrides, rename a sidecar with its parent, and handle a sidecar without a parent on its own.
  [verified]
- **GMI-035** Where `check_disk_space` is set, if the destination lacks free space for the
  planned copies, then the importer shall fail before copying. [verified]

## Copying

- **GMI-040** The importer shall copy with a pool of `workers` goroutines, four when `workers` is
  0. [verified]
- **GMI-041** The importer shall check that each copy wrote the source's byte count, set the
  copy's timestamps from the source, and remove a partial copy when copying fails. [verified]
- **GMI-042** The importer shall not checksum files after copying them. [verified]
- **GMI-043** While `dry_run` is set, the importer shall report what it would copy and clean up
  without changing any file. [verified]
- **GMI-044** If any copy or deletion fails, then the importer shall finish the others, report
  every error, and exit non-zero. [verified]
- **GMI-045** While verbose output is on and stdout is not a terminal, the importer shall print
  plain progress lines, and it shall print no progress while verbose output is off. [verified]

## Source cleanup and ejection

- **GMI-050** Where `delete_originals` is set, the importer shall delete each imported original
  and its configured sidecars after the copies succeed. [verified]
- **GMI-051** The importer shall delete recorded source artifacts only when `delete_originals` is
  set and planning, copying and original deletion all succeeded, and preserve them otherwise.
  [verified]
- **GMI-052** If a recorded cleanup target lies outside the source or is a symbolic link, then
  the importer shall refuse to delete it. [verified]
- **GMI-053** If artifact cleanup fails, then the importer shall report every failure and exit
  non-zero. [verified]
- **GMI-054** When ejecting a source on macOS, the importer shall run `diskutil eject <source>`;
  on Linux, `udisksctl unmount`. [verified]
- **GMI-055** Where `auto_eject` is set, the importer shall eject the source only after planning,
  copying, original deletion and artifact cleanup all succeeded.
  unverified: ejecting needs a real removable device; the order is the early returns in
  `importMedia`, which no test drives to the eject step.

## Removable volumes

- **GMI-060** When run as `volumes list`, the importer shall list every mounted removable volume
  with a row ID, label, mount path, whether its label is saved and its destination. [verified]
- **GMI-061** When run as `volumes add LABEL|ID [--dest DEST]`, the importer shall save a
  currently mounted volume's label to the config, keep an existing destination when `--dest` is
  omitted, refuse a label that is not mounted, and refuse a config with unknown keys before
  changing it. [verified]
- **GMI-062** When `--source` is omitted and removable volume labels are configured, the importer
  shall import every mounted removable volume whose label is saved, including several with the
  same label, each to its label's destination or the global one. [verified]
- **GMI-063** If importing one configured removable volume fails, then the importer shall still
  import the others and exit non-zero. [verified]
- **GMI-064** On macOS, the importer shall treat a mount as a removable volume when it has
  `MNT_REMOVABLE`, or is a local `/dev/disk*` mount directly under `/Volumes`, and never a
  network share or system volume. [verified]
- **GMI-065** On Linux, the importer shall discover removable volumes from
  `/proc/self/mountinfo`, `/sys/dev/block` and `/dev/disk/by-label`.
  unverified: needs real removable block devices; the discovery is exercised only by hand.

## Releases

- **GMI-080** When CI succeeds for a push to main, the release workflow shall publish
  `gomediaimport-<version>-{darwin,linux}-{arm64,amd64}` binaries with `SHA256SUMS` and a
  provenance attestation signed by `.github/workflows/release.yml` on `refs/heads/main`, and
  verify the published release. [verified]
- **GMI-081** The release workflow shall choose the next patch after the highest tag of the
  series in `VERSION`, or `.0` for a new series, finish the release of an already tagged commit,
  and refuse a version not higher than every tag or a commit that lacks the highest tag.
  [verified]
- **GMI-082** The Go module path shall carry the `/vN` suffix of the major version in `VERSION`,
  so `go install ...@latest` resolves to the newest release. [verified]
