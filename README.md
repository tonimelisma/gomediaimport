# gomediaimport

gomediaimport is a CLI tool that imports and organizes pictures and videos from SD cards, external drives, or any source directory. It simplifies the process of managing and organizing your media files with features like duplicate detection, date-based organization, and concurrent copying.

## Features

- Import media files from any source directory
- Remember removable volume labels and import all matching mounted volumes with one command
- Concurrent file copying with configurable worker count
- Duplicate detection with optional xxHash64 verification for apparent duplicates
- Optional file organization into date-based subdirectories (`YYYY/MM`)
- Optional file renaming by creation date and time (`YYYYMMDD_HHMMSS`), with deterministic same-second suffixes based on original filename order
- Image EXIF/XMP and MP4/MOV-family video metadata extraction for accurate creation dates
- Sidecar file handling (XMP, THM, CTG, etc.) with configurable actions
- Dry-run mode for safe previewing
- Idempotent: safe to re-run without duplicating files

## Installation

```bash
go install github.com/tonimelisma/gomediaimport/cmd/gomediaimport@latest
```

Or clone and build locally:

```bash
git clone https://github.com/tonimelisma/gomediaimport.git
cd gomediaimport
go install -ldflags "-X main.version=2.0.0" ./cmd/gomediaimport
```

This installs gomediaimport into your `$GOPATH/bin` directory. Ensure it's in your PATH.

To embed the version number, pass it via `-ldflags` as shown above. Without it, `--version` will print `dev`.

## Usage

### Command-line options

```bash
gomediaimport [--source SOURCE] [--dest DEST] [--config CONFIG]
  [--organize-by-date] [--rename-by-date-time] [--checksum-duplicates]
  [--no-checksum-duplicates] [-v] [--dry-run] [--skip-thumbnails] [--delete-originals] [--auto-eject]
  [--check-disk-space] [--sidecar-default ACTION] [--workers N] [--version]

gomediaimport volumes list [--config CONFIG]
gomediaimport volumes add LABEL [--dest DEST] [--config CONFIG]
gomediaimport volumes add ID [--dest DEST] [--config CONFIG]
```

- `--source SOURCE`: Source directory for a one-off import (optional if set in config file and no saved removable volumes are configured)
- `--dest DEST`: Destination directory for imported media (default: `~/Pictures`)
- `--config CONFIG`: Path to config file. The default platform-specific path is shown in `--help`.
- `--organize-by-date`: Organize files into `YYYY/MM` subdirectories by creation date
- `--rename-by-date-time`: Rename files to `YYYYMMDD_HHMMSS` format based on creation date. Same-second collisions use `_001`, `_002`, etc. in natural original filename order.
- `--checksum-duplicates`: Use xxHash64 checksums for duplicate detection (default)
- `--no-checksum-duplicates`: Disable checksum duplicate verification and use file size/timestamp matching only
- `-v, --verbose`: Enable verbose output with progress information
- `-q, --quiet`: Suppress all non-error output (forces verbose off)
- `--dry-run`: Preview what would happen without making any changes
- `--skip-thumbnails`: Skip thumbnail directories in source data (e.g. video thumbnails)
- `--delete-originals`: Delete original files after successful import
- `--auto-eject`: Eject the source drive after a fully successful import (default: `false`). Uses `diskutil eject` on macOS, `udisksctl unmount` on Linux.
- `--check-disk-space`: Check for sufficient free disk space on the destination before importing (default: `true`). Use `--check-disk-space=false` to disable.
- `--sidecar-default ACTION`: Default action for sidecar file types: `ignore`, `copy`, or `delete` (default: `delete`)
- `--workers N`: Number of concurrent copy workers (default: 4)
- `--version`: Print version and exit
- `volumes list`: List currently mounted removable volumes
- `volumes add LABEL`: Save a currently mounted removable volume label to the config
- `volumes add ID`: Save the label from the numbered row shown by `volumes list`
- `volumes add ... --dest DEST`: Set a destination for that saved label; otherwise the global destination is used

### Examples

```bash
# Import media with default settings
gomediaimport --source /media/sdcard

# Import using source directory from config file
gomediaimport

# List removable volumes currently mounted
gomediaimport volumes list

# Save a currently mounted removable volume label using the default destination
gomediaimport volumes add SOFIA

# Save the first listed removable volume with its own destination
gomediaimport volumes add 1 --dest "/Users/me/Pictures/Sofia"

# Import all configured removable volume labels currently mounted
gomediaimport

# Import media and organize by date into YYYY/MM subdirectories
gomediaimport --organize-by-date --source /media/sdcard

# Import, rename by date/time, and delete originals
gomediaimport --rename-by-date-time --delete-originals --source /media/sdcard

# Perform a dry run without making changes
gomediaimport --dry-run --source /media/sdcard

# Disable checksum duplicate verification for faster planning
gomediaimport --no-checksum-duplicates --source /media/sdcard
```

## Configuration

gomediaimport can be configured using a YAML configuration file. The default config location is platform-idiomatic:

- **macOS**: `~/Library/Application Support/gomediaimport/config.yaml`
- **Linux**: `~/.config/gomediaimport/config.yaml`

You can specify a different path using `--config`.

An example configuration file [`config.yaml`](config.yaml) is provided in the root of this repository.

Set `checksum_duplicates: false` to disable checksum duplicate verification and use size/timestamp-only matching.

### Removable volumes

Saved removable volumes are configured by label. Labels are selectors, not unique identities: if multiple currently mounted removable volumes have the same saved label, gomediaimport imports all of them. The source directory is computed from each volume's current mount path at runtime.

```yaml
destination_directory: "/Users/me/Pictures/Camera Roll"

removable_volumes:
  SOFIA: {}
  "4152150790":
    destination_directory: "/Users/me/Pictures/Camera 4152150790"
```

In this example, mounted removable volumes labeled `SOFIA` import to the global `destination_directory`. Mounted removable volumes labeled `4152150790` import to their volume-specific destination.

## Supported File Types

gomediaimport supports a wide range of media file types:

### Processed Pictures
- JPEG (.jpg, .jpeg, .jpe, .jif, .jfif, .jfi)
- JPEG 2000 (.jp2, .j2k, .jpf, .jpm, .jpg2, .j2c, .jpc, .jpx, .mj2)
- JPEG XL (.jxl)
- PNG (.png)
- GIF (.gif)
- BMP (.bmp)
- TIFF (.tiff, .tif)
- PSD (.psd)
- EPS (.eps)
- SVG (.svg)
- ICO (.ico)
- WebP (.webp)
- HEIF (.heif, .heifs, .heic, .heics, .avci, .avcs, .hif)

### Raw Pictures
- Various RAW formats (.arw, .cr2, .cr3, .crw, .dng, .erf, .kdc, .mrw, .nef, .orf, .pef, .raf, .raw, .rw2, .sr2, .srf, .x3f)

### Videos
- MP4 (.mp4), AVI (.avi), MOV (.mov), WMV (.wmv), FLV (.flv)
- MKV (.mkv), WebM (.webm), OGV (.ogv), M4V (.m4v)
- 3GP (.3gp), 3G2 (.3g2), ASF (.asf), VOB (.vob)
- MTS (.mts, .m2ts)

For embedded timestamps, gomediaimport uses `videometa` on `.mp4`, `.mov`, `.m4v`, `.3gp`, and `.3g2` files to read QuickTime-native metadata plus supported vendor metadata routes and container config. Other video containers still import normally, but creation time falls back to filesystem mtime when no supported embedded timestamp is available.

### Raw Videos
- Various RAW video formats (.braw, .r3d, .ari)

### Sidecar Files
- XMP (.xmp) — copied by default
- SRT (.srt) — copied by default
- THM (.thm), CTG (.ctg), AAE (.aae), LRF (.lrf), MPL (.mpl), CPI (.cpi) — deleted by default

Use `--sidecar-default` to change the default action, or configure per-extension overrides in the config file.

### Adding file types

File type support is defined in `media_types.go`. Pull requests for missing file types are welcome.

## How It Works

1. **Configuration**: Loads settings from built-in defaults, then the YAML config file, then CLI arguments. If configured removable volume labels exist and `--source` is not provided, gomediaimport discovers currently mounted removable volumes and imports every matching label.

2. **Enumeration**: Scans the source directory recursively, identifying media files by extension and extracting creation dates from image EXIF/XMP or supported MP4/MOV-family video metadata (falls back to file modification time when no supported embedded timestamp is available).

3. **Destination Planning**: Determines each file's destination path based on organization and renaming settings. Date-time rename imports sort files by capture time and natural original filename order first, so same-second rename collisions receive deterministic suffixes. Detects duplicates using an O(1) size+timestamp index, with xxHash64 checksum verification enabled by default.

4. **Concurrent Copying**: Copies files using a worker pool (default 4 workers) with size-interleaved scheduling for balanced load. Each copy checks that the written byte count matches the source size and then closes the destination file.

5. **Cleanup**: Optionally deletes original files after successful copy. Can eject the source drive (macOS via `diskutil`, Linux via `udisksctl`).

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a branch for your feature or fix
3. Write tests to cover your changes
4. Ensure all tests pass (`go test -race ./cmd/gomediaimport`)
5. Submit a pull request

## License

gomediaimport is released under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

This software was designed by Toni Melisma and written by [Claude](https://claude.ai/). Writing the initial version cost exactly $5.00 in LLM costs.
