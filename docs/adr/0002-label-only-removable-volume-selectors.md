# 0002. Removable volumes are selected by label only

- Status: accepted
- Date: 2026-05-26

## Context and problem

Plain `gomediaimport` should import whatever camera cards are plugged in, without a `--source`.
A card's mount path changes between machines and plugs, so the config needs a stable selector.
Serves GMI-060 to GMI-065.

## Decision drivers

- Works for the same card on macOS and Linux, wherever it mounts.
- Nothing in the config that changes between plugs.
- Discovery without cgo, so release binaries cross-compile.

## Considered options

### Store mount paths

- Good: trivial to resolve.
- Bad: paths change (`/Volumes/CAMERA 1`, `/media/<user>/CAMERA`).
- Failure modes: the import silently finds nothing, or imports the wrong disk.

### Store unique identities (volume UUID, serial)

- Good: unambiguous.
- Bad: platform-specific, awkward to read and edit, and several cards from one camera share
  nothing a user recognizes.
- Failure modes: a reformatted card gets a new identity and stops importing.

### Store labels, resolve mount paths at runtime

- Good: readable, survives reformatting in the same camera, the same on both platforms.
- Bad: labels are not unique.
- Failure modes: two mounted cards with one label; both are imported, which is the wanted
  behavior for cameras that label every card alike.

## Decision

`removable_volumes` stores labels only, each with an optional `destination_directory`. The
source path is computed from the currently mounted removable volumes each run; every mounted
volume with a saved label is imported. `source_directory` stays supported, but configured
removable volumes are used when present and `--source` is omitted. Discovery is no-cgo: macOS
uses `Getfsstat` and `getattrlist`, Linux `/proc/self/mountinfo`, `/sys/dev/block` and
`/dev/disk/by-label`.

## Consequences

- `volumes list` and `volumes add` manage the labels; the config never holds a mount path.
- A non-camera disk that happens to carry a saved label would be imported.

## Confirmation

`TestImportConfiguredRemovableVolumes`, `TestImportConfiguredRemovableVolumesContinuesAfterFailure`,
`TestAddRemovableVolume`, `TestListRemovableVolumes` (GMI-060 to GMI-063) and, on macOS,
`TestDarwinIsRemovableVolumeCandidate` (GMI-064).

## Retirement proof

Reports of wrong disks imported because of shared labels.

## Reversal plan

Add an optional identity field next to the label that narrows the match; existing label-only
entries keep working.
