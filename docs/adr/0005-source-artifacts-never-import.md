# 0005. Source artifacts never import and are cleaned only with the originals

- Status: accepted
- Date: 2026-06-20

## Context and problem

Cards carry files that look like media but are not: trashed JPGs in `.Trashes`, Sony XAVC
thumbnails and XML companions, macOS AppleDouble `._*` files. Imported, they pollute the library
(trashed photos come back); left on the card while originals are deleted, they pile up. Serves
GMI-022 and GMI-051 to GMI-053.

## Decision drivers

- Nothing but real media reaches the destination.
- Cleanup never deletes outside the source or through a symlink.
- One switch controls source cleanup.

## Considered options

### Classify artifacts by extension like media, with a skip option

- Good: simple.
- Bad: needs an option per artifact kind (the removed `skip_thumbnails`); misclassifies `._*`.
- Failure modes: trashed photos restored into the library.

### Recognize exact artifact paths before classification

- Good: artifacts never enter media classification; cleanup knows exactly what it may delete.
- Bad: each new artifact kind needs code.
- Failure modes: an unrecognized artifact kind is imported as media until added.

## Decision

Enumeration recognizes top-level system trash, exact Sony `[PRIVATE/]M4ROOT/THMBNL` directories,
`[PRIVATE/]M4ROOT/CLIP/*.XML` and AppleDouble files, and records them as cleanup targets instead
of classifying them. `delete_originals` is the only cleanup control: with it off they are
preserved; with it on they are deleted after planning, copying and original deletion succeed.
Cleanup refuses targets outside the source and symlinks.

## Consequences

- `skip_thumbnails` is gone; the strict config rejects it.
- Dry runs and verbose output list cleanup targets separately from media.

## Confirmation

`TestSourceArtifactPathClassification`, `TestEnumerateFilesSeparatesSourceArtifacts` (GMI-022),
`TestImportMediaSourceArtifactLifecycle` (GMI-051), `TestCleanupSourceArtifactsRejectsUnsafeTargets`
(GMI-052).

## Retirement proof

A camera vendor or OS artifact that cannot be recognized by path.

## Reversal plan

Make the recognized paths configurable; the cleanup ordering stays.
