# 0001. Idempotent imports derived from the filesystem, with no state file

- Status: accepted
- Date: 2026-05-26

## Context and problem

An import is often re-run on the same card, after a failure or with more photos added. The tool
must never copy a file twice, and must not depend on anything that can drift from what is on
disk. Serves GMI-032, GMI-033 and GMI-042.

## Decision drivers

- Re-running is always safe: no duplicate copies, no lost files.
- Nothing to corrupt, migrate or reset between runs.
- Planning stays fast on large destinations.

## Considered options

### A state file or database of imported files

- Good: a repeated import is a lookup, with no hashing.
- Bad: drifts from the disk when files are moved, deleted or imported by another tool; needs a
  location, a format and migrations.
- Failure modes: a stale entry skips a file that is no longer in the destination; a lost file
  re-imports everything.

### Derive everything from the filesystem on every run

- Good: the destination itself is the record; nothing can drift.
- Bad: every run indexes the destination.
- Failure modes: two different files with the same size and timestamp, which checksums catch.

## Decision

Derive all state from the filesystem each run. Duplicate candidates come from an O(1)
size-and-timestamp index of the destination and of earlier files in the batch; by default a
candidate counts as a duplicate only when its xxHash64 checksum also matches
(`checksum_duplicates: false` / `--no-checksum-duplicates` uses size and timestamp alone).
Copies are not checksummed after transfer: a copy is checked by its written byte count and an
explicit `Close()` instead.

## Consequences

- Safe to re-run at any point; there is no state to clean up.
- Checksums are computed only for apparent duplicates, so the common case reads each file once.
- A copy corrupted in transit with the right byte count is not detected by the importer.

## Confirmation

`TestIsDuplicate`, `TestIsDuplicateInPreviousFiles`, `TestCopyFilesSkipsPreExisting` (GMI-033)
and `TestCopyFilesDoesNotVerifyCopiedFilesWithChecksum` (GMI-042).

## Retirement proof

A measured need, such as destinations too large to index per run, shown with timings.

## Reversal plan

Add a cache keyed by path, size and mtime that only accelerates the index, never replaces it;
the filesystem stays the source of truth.
