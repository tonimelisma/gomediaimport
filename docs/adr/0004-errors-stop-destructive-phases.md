# 0004. Every error is reported, and any error stops the destructive phases

- Status: accepted
- Date: 2026-06-20

## Context and problem

An import copies, then may delete originals, clean source artifacts and eject the card. If a
copy fails silently and deletion still runs, photos are lost. Serves GMI-044, GMI-050 to
GMI-055.

## Decision drivers

- Never delete a source whose copy did not succeed.
- Report every failure, not only the first.
- Exit non-zero when anything failed, so scripts notice.

## Considered options

### Best effort: log and continue through every phase

- Good: the most work gets done in one run.
- Bad: deletion can follow a failed copy.
- Failure modes: lost originals.

### Accumulate errors per phase, and stop before the next destructive phase

- Good: a phase finishes every file it can and reports all failures; nothing later runs.
- Bad: one bad file blocks deletion and ejection for the whole card until it is fixed.
- Failure modes: a card stays mounted with originals in place, which is safe.

## Decision

`copyFiles()`, `deleteOriginalFiles()` and `cleanupSourceArtifacts()` finish what they can and
return every error with `errors.Join()`. A planning, copy, deletion or cleanup error returns from
`importMedia` before the next phase, so originals are deleted only after copies succeed,
artifacts only after deletion succeeds, and ejection only after all of them. Non-fatal warnings
(checksum or timestamp-setting failures) go to stderr, never swallowed; the process exits 1 on
any error.

## Consequences

- A partial failure is always visible and leaves the source intact.
- Each phase's error handling must keep accumulating instead of returning early.

## Confirmation

`TestCopyFilesAccumulatesCopyError`, `TestDeleteOriginalFilesReturnsError` (GMI-044) and the
`TestImportMedia*PreservesSourceArtifacts` tests (GMI-051). The eject ordering (GMI-055) is not
covered by a test.

## Retirement proof

None foreseen: this protects the user's only copy of their media.

## Reversal plan

Not planned; a "force" mode would be a separate, explicit flag.
