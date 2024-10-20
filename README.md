# gomediaimport

gomediaimport is a tool designed to automatically import pictures and videos from SD cards or other volumes. It simplifies the process of managing and organizing your media files. It can automatically import media from inserted removable volumes on macOS.

## Features

- Automatic import from SD cards or other volumes
- Configurable import settings
- Support for various media file types
- Optional file organization by date
- Duplicate detection

## Installation

```bash
go install ./cmd/gomediaimport
```

This will install gomediamport into your `~/go/bin` directory. Ensure it's in your PATH.

## Usage

### Basic command-line usage

```bash
gomediaimport [--dest DEST] [--config CONFIG] [--organize-by-date]
[--rename-by-date-time] [--checksum-duplicates] [--checksum-imports]
[-v] [--dry-run] [--skip-thumbnails] [--delete-originals]
SOURCE_DIR
```

- `SOURCE_DIR`: Source directory for media files (required)
- `--dest DEST`: Destination directory for imported media
- `--config CONFIG`: Path to config file
- `--organize-by-date`: Organize files by date
- `--rename-by-date-time`: Rename files by date and time
- `--checksum-duplicates`: Use checksums to identify duplicates (slow, otherwise uses size and date/time)
- `--checksum-imports`: Calculate checksums for imported files (slow, otherwise uses size and date/time)
- `-v, --verbose`: Enable verbose output
- `--dry-run`: Perform a dry run without making changes
- `--skip-thumbnails`: Skip thumbnails in source data (e.g. video thumbnails)
- `--delete-originals`: Delete original files after successful import

### Examples

```bash
# Import media with default settings
gomediaimport /Volumes/SD_CARD

# Import media and organize by date
gomediaimport --organize-by-date /Volumes/SD_CARD

# Perform a dry run without making changes
gomediaimport --dry-run /Volumes/SD_CARD
```

## Configuration

gomediaimport can be configured using a YAML configuration file. By default, the program looks for a configuration file at `~/.gomediaimportrc`, but you can specify a different path using the `--config` command-line option.

An example configuration file [`gomediaimportrc`](gomediaimportrc) is provided in the root of this repository. You can copy this file to `~/.gomediaimportrc` (or your preferred location) and modify it according to your needs.

Note that command-line arguments will override settings in the configuration file.

## Automatic macOS Launch

You can automatically launch gomediaimport when a volume is inserted on macOS. You will need to install the separate `gomediaimport-launchagent` for that. It will keep a log of all inserted removable volumes, and run gomediaimport for each when they're inserted.

To install and enable the macOS launch agent:
* Open the separate Xcode project, build the binary and place it in your directory of choosing
* Take the example file [net.melisma.gomediamport-launchagent.plist](gomediaimport-launchagent/net.melisma.gomediaimport-launchagent.plist) and copy it into `~/Library/Launch Agents`
* Edit the file and change ProgramArguments to refer to the path you placed the binary in
* Run `launchctl load ~/Library/LaunchAgents/net.melisma.gomediamport-launchagent.plist`

You can `tail -f ~/.gomediaimport-launchagent.log` to see the log of inserted volumes.

## Supported File Types

gomediaimport supports a wide range of media file types, categorized as follows:

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
- MP4 (.mp4)
- AVI (.avi)
- MOV (.mov)
- WMV (.wmv)
- FLV (.flv)
- MKV (.mkv)
- WebM (.webm)
- OGV (.ogv)
- M4V (.m4v)
- 3GP (.3gp)
- 3G2 (.3g2)
- ASF (.asf)
- VOB (.vob)
- MTS (.mts, .m2ts)

### Raw Videos
- Various RAW video formats (.braw, .r3d, .ari)

### Other formats

gomediaimport uses the [github.com/evanoberholster/imagemeta](`github.com/evanoberholster/imagemeta`) package for metadata extraction, but it's compatible with any media files matching the supported extensions. If you need support for additional file types, you can easily add them to the `media_types.go` file.

We welcome Pull Requests if you find any missing file types or have improvements to suggest!

## How It Works
)
gomediaimport is designed to efficiently import and organize media files. Here's an overview of its operation:

1. **Configuration**: The program starts by loading configuration from a YAML file (default: `~/.gomediaimportrc`) and command-line arguments. The `setDefaults` function in `main.go` sets initial values, which can be overridden by the config file and CLI arguments.

2. **File Enumeration**: The `enumerateFiles` function in `file_operations.go` scans the source directory for media files, identifying their types based on extensions defined in `media_types.go`.

3. **Metadata Extraction**: For each file, the program attempts to extract creation date and time from metadata using the `extractCreationDateTimeFromMetadata` function in `metadata.go`. It utilizes the `github.com/evanoberholster/imagemeta` package for this purpose.

4. **Duplicate Detection**: The `isDuplicate` function in `file_operations.go` checks for duplicate files based on size and optionally CRC32 checksums.

5. **File Organization**: If enabled, files are organized into subdirectories based on their creation date.

6. **File Renaming**: If the `rename_by_date_time` option is set, files are renamed according to their creation date and time.

7. **File Copy**: The `copyFiles` function in `import.go` handles the actual file copying process, creating necessary directories and handling naming conflicts.

8. **Original File Deletion**: If the `delete_originals` option is set, the `deleteOriginalFiles` function in `import.go` removes the original files after successful import.

9. **Logging and Verbose Output**: Throughout the process, if verbose mode is enabled, the program provides detailed information about its operations.

gomediaimport supports a wide range of file types for both images and videos, as defined in the `fileTypes` slice in `media_types.go`. It's designed to be extensible, allowing easy addition of new file types.

The program uses efficient file operations and provides options for checksum-based duplicate detection to ensure data integrity. It also includes a dry-run mode for testing configurations without making actual changes to the file system.

## Contributing

Contributions to gomediaimport are welcome! Here's how you can contribute:

1. Fork the repository
2. Create a new branch for your feature or bug fix
3. Make your changes and write tests to cover your contributions
4. Ensure all tests pass
5. Submit a pull request with a clear description of your changes

Please remember to include unit tests that cover your contributions.

## License

gomediaimport is released under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

This software was designed by Toni Melisma and written by [Claude 3.5 Sonnet](https://claude.ai/) with the assistance of [continue.dev](https://continue.dev/). Writing this program cost exactly $5.00 in LLM costs.

## Roadmap
- Eject removable volume after import
- Handle metadata sidecar XML files for videos (copy with videos, delete originals after import)
- Delete thumbnails after import
- Logging
- Multithreading
- Verify integrity of all copied files
- Handle multiple import directories