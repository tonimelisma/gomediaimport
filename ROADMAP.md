# Roadmap

## Planned Features

- Video metadata extraction (currently falls back to mtime)
- Structured logging (replace ad-hoc fmt.Printf)
- Multithreading for copy operations
- Upgrade yaml.v2 to yaml.v3
- Handle metadata sidecar XML files for videos (copy with videos, delete originals after import)
- Delete thumbnails after import
- Handle multiple import directories
- Don't prompt for source directory if defined in config file

## Design Decisions

### xxHash64 for duplicate detection

Chosen for speed (faster than CRC32) and 64-bit collision space. In this non-adversarial context — comparing files you own on local storage — accidental collisions at 2^64 are negligible. SHA-256 would add no practical benefit while roughly halving throughput.

### No post-copy verification

Copying already validates that the byte count written matches the source file size, and `Sync()` flushes to stable storage. Adding a checksum verification pass would re-read every copied file (doubling I/O) for no practical gain. Silent corruption during a buffered, synced copy on a modern filesystem is not a realistic failure mode.
