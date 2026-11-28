# 0007. Releases are built, attested and published from CI-green main only

- Status: accepted
- Date: 2026-11-27

## Context and problem

Releases were made by hand (`gh release create` as the last step of every task, deleting and
re-creating the tag to fix anything), with no binaries. That depends on the person releasing,
can tag a commit CI never passed, and offers nothing to install without a Go toolchain; the
module path also lacked the `/v3` suffix, so `go install ...@latest` installed v1.9.0. Serves
GMI-080 to GMI-082.

## Decision drivers

- Every merge to main ships, with no manual step.
- Binaries can be verified as built from this repository's main by this workflow.
- The version is computed, never typed, and a new series is an ordinary reviewed change.

## Considered options

### Manual tags and releases

- Good: full control over when to release.
- Bad: forgotten or wrong tags; commits released without CI; no binaries.
- Failure modes: a release of an untested commit; a re-created tag that installers have cached.

### A workflow on tag pushes

- Good: familiar.
- Bad: anyone who can push a tag can release any commit; the version is still typed by hand.
- Failure modes: a tag on a branch commit gets an attestation that looks like a main release.

### A workflow triggered by CI succeeding on main

- Good: only tested main commits ship; the workflow picks the version and creates the tag.
- Bad: every merge is a release, including docs-only ones.
- Failure modes: CI runs finishing out of order (refused: a commit lacking the highest tag is
  never released).

## Decision

`.github/workflows/release.yml` runs on `workflow_run` of CI succeeding for a push to main (and by
dispatch on main, to finish a failed run). `script/release-version` picks the next patch after
the highest tag of the series in `VERSION`, or `.0` for a new series. The workflow builds static
`darwin` and `linux` `amd64`/`arm64` binaries stamped with `-X main.version`, writes
`SHA256SUMS`, attests them with `actions/attest-build-provenance`, publishes with generated
notes, and verifies the published release with `gh attestation verify`. The module path carries
the major version suffix (`/v3`).

## Consequences

- Nobody tags or publishes by hand; a minor or major series starts by editing `VERSION` in a PR.
- A major series bump also changes the module path suffix (a test enforces it).
- The signing identity is `.github/workflows/release.yml` on `refs/heads/main`: renaming the file
  or releasing from another ref changes what verifiers must accept.

## Confirmation

`script/release-version_test.sh` (GMI-081), the release workflow's verify step (GMI-080) and
`TestModulePathMatchesVersionSeries` (GMI-082).

## Retirement proof

A packaging channel (Homebrew tap, distro packages) that needs a different release shape.

## Reversal plan

Disable the workflow and tag by hand; published releases and attestations stay valid.
