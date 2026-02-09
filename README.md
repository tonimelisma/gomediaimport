# gomediaimport

gomediaimport is a CLI tool that imports and organizes pictures and videos from SD cards, external drives, or any source directory. It simplifies the process of managing and organizing your media files with features like duplicate detection, date-based organization, and concurrent copying.

## Features

- Import media files from any source directory
- Concurrent file copying with configurable worker count
- Duplicate detection using file size, timestamps, and optional xxHash64 checksums
- Optional file organization into date-based subdirectories (`YYYY/MM`)
- Optional file renaming by creation date and time (`YYYYMMDD_HHMMSS`)
- EXIF and video metadata extraction for accurate creation dates
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
go install ./cmd/gomediaimport
```

This installs gomediaimport into your `$GOPATH/bin` directory. Ensure it's in your PATH.

## Usage

### Command-line options

```bash
gomediaimport [--dest DEST] [--config CONFIG] [--organize-by-date]
  [--rename-by-date-time] [--checksum-duplicates]
  [-v] [--dry-run] [--skip-thumbnails] [--delete-originals]
  [--auto-eject-macos] [--sidecar-default ACTION]
  [--workers N] [--version]
  [SOURCE_DIR]
```

- `SOURCE_DIR`: Source directory for media files (optional if set in config file)
- `--dest DEST`: Destination directory for imported media (default: `~/Pictures`)
- `--config CONFIG`: Path to config file (default: `~/.gomediaimportrc`)
- `--organize-by-date`: Organize files into `YYYY/MM` subdirectories by creation date
- `--rename-by-date-time`: Rename files to `YYYYMMDD_HHMMSS` format based on creation date
- `--checksum-duplicates`: Use xxHash64 checksums for duplicate detection (slower but more accurate; otherwise uses file size and timestamp)
- `-v, --verbose`: Enable verbose output with progress information
- `--dry-run`: Preview what would happen without making any changes
- `--skip-thumbnails`: Skip thumbnail directories in source data (e.g. video thumbnails)
- `--delete-originals`: Delete original files after successful import
- `--auto-eject-macos`: On macOS, eject the source drive after a fully successful import (default: `false`)
- `--sidecar-default ACTION`: Default action for sidecar file types: `ignore`, `copy`, or `delete` (default: `delete`)
- `--workers N`: Number of concurrent copy workers (default: 4)
- `--version`: Print version and exit

### Examples

```bash
# Import media with default settings
gomediaimport /media/sdcard

# Import using source directory from config file
gomediaimport

# Import media and organize by date into YYYY/MM subdirectories
gomediaimport --organize-by-date /media/sdcard

# Import, rename by date/time, and delete originals
gomediaimport --rename-by-date-time --delete-originals /media/sdcard

# Perform a dry run without making changes
gomediaimport --dry-run /media/sdcard

# Use checksums for more accurate duplicate detection
gomediaimport --checksum-duplicates /media/sdcard
```

## Configuration

gomediaimport can be configured using a YAML configuration file. By default, the program looks for `~/.gomediaimportrc`, but you can specify a different path using `--config`.

An example configuration file [`gomediaimportrc`](gomediaimportrc) is provided in the root of this repository. Copy it to `~/.gomediaimportrc` and modify it according to your needs.

Command-line arguments always override settings in the configuration file. Configuration precedence: CLI flags > YAML config file > built-in defaults.

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

1. **Configuration**: Loads settings from built-in defaults, then the YAML config file, then CLI arguments.

2. **Enumeration**: Scans the source directory recursively, identifying media files by extension and extracting creation dates from EXIF/video metadata (falls back to file modification time).

3. **Destination Planning**: Determines each file's destination path based on organization and renaming settings. Detects duplicates using an O(1) size+timestamp index, with optional checksum verification.

4. **Concurrent Copying**: Copies files using a worker pool (default 4 workers) with size-interleaved scheduling for balanced load. Files are synced to disk and verified for completeness.

5. **Cleanup**: Optionally deletes original files after successful copy. On macOS, can eject the source drive.

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
