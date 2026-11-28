# CLAUDE.md

## What this is

gomediaimport imports and organizes photos and videos from SD cards, removable volumes or any
directory into a date-organized library, detecting duplicates and cleaning camera artifacts. A
single-package Go CLI in `cmd/gomediaimport/` (module `github.com/tonimelisma/gomediaimport/v3`),
released as attested darwin and linux binaries for amd64 and arm64. Architecture:
[docs/architecture.md](docs/architecture.md).

## Working style

- When it is clear what is wanted, do it. Ask only for what only the user can do, or for a real
  product choice nobody has made yet.
- Don't end a turn with a summary announcing the next step, an offer to continue, or a list of
  decisions that block nothing: take the next step.
- Stop only when nothing can move forward without the user, or when what blocks you is
  deliberately protected (a permission, a ruleset, a secret). Then say exactly what is needed.
- Still confirm destructive or irreversible actions outside the normal flow (deleting data,
  rewriting history, changing settings); the normal flow (branch, commit, PR, merge, release) is
  not one of them.

## Product invariants

- Re-running an import never copies a file twice, and all state is derived from the filesystem
  each run; there is no state file. ([ADR 0001](docs/adr/0001-idempotent-imports-without-state.md))
- Removable volumes are selected by label only; mount paths are resolved at runtime and every
  mounted volume with a saved label is imported.
  ([ADR 0002](docs/adr/0002-label-only-removable-volume-selectors.md))
- Unknown or removed config keys are errors, flags beat the config file (including
  `--flag=false`), and the code keeps no global state.
  ([ADR 0003](docs/adr/0003-strict-config-with-explicit-overrides.md))
- No original is deleted unless its copy succeeded, no artifact unless original deletion
  succeeded, and no source is ejected unless everything before succeeded; every error is
  reported and the exit code is 1. ([ADR 0004](docs/adr/0004-errors-stop-destructive-phases.md))
- Trash, Sony XAVC support files and AppleDouble files never import, and are deleted only under
  `delete_originals`, never outside the source or through a symlink.
  ([ADR 0005](docs/adr/0005-source-artifacts-never-import.md))
- Only `.github/workflows/release.yml`, running on CI-green main, tags, releases or attests.
  ([ADR 0007](docs/adr/0007-attested-release-from-ci-green-main.md))

## Engineering philosophy

- No compatibility shims. When a structure is wrong, replace it and migrate its callers in the
  same change.
- The smallest correct design. Simplicity first; measure before optimizing, never optimize on a
  hunch.
- One owner per fact, per mutation and per side effect. Two places that know the same thing will
  disagree.
- Abstractions only at real boundaries (a process, a network, a storage format, a platform), not
  in anticipation.
- Don't guess and don't fabricate: an unknown API, path, value or result is looked up, measured or
  reported as unknown.
- You own the repository. Nothing is "pre-existing" or someone else's problem, and red CI is never
  waived: it is fixed first.
- Comments and docs describe current behavior and why it is that way, never its history.

## Engineering practices

- Test first. A change in behavior starts with a test that fails without it.
- Every bug, CI failure and review finding gets a regression test, seen failing before the fix;
  quote its red output in the PR.
- The instrumentation flywheel, every increment:
  - Never guess a bug's cause. If the logs and metrics can't explain it, add the instrumentation
    first, so this occurrence or the next one is diagnosable, then fix.
  - Never guess at performance: measure, keep a baseline, optimize, and report before and after.
  - Even when nothing is blocked, review the instrumentation on every path you touched: add what
    was missing, remove what is noise.
- Tests are hermetic: temporary directories and fixtures, never the developer's real data, config
  or accounts.
- Go: `gofmt`, `go vet` for both release platforms, golangci-lint (`.golangci.yml`, which
  includes staticcheck) and govulncheck, all through `script/verify`. CI runs it on Linux and
  macOS, so each platform's files (`*_darwin.go`, `*_linux.go`) are tested and linted on it.
- `run(osArgs)` is the testable entry point: tests pass their own arguments and an
  `emptyConfigFile(t)`, and touch only `t.TempDir()` paths.
- Every action in `.github/workflows/` is pinned to a commit SHA with a `# vX.Y.Z` comment, and
  every tool in `run:` or `script/verify` to an exact version.

## Commands

```sh
script/verify                                          # everything CI checks; run before every push
go build ./cmd/gomediaimport                           # build
go test -race ./...                                    # full test suite
go test -run TestEnumerateFiles ./cmd/gomediaimport    # one test
gofmt -w .                                             # format
script/process-check                                   # requirements, Validates tags and doc links agree
script/stage start | pre-merge <PR> | post-merge <PR>
```

## Where docs live

- `docs/index.md` maps every doc. Requirements live in `docs/requirements.md`: stable IDs, one
  EARS sentence each, current behavior only, each `[verified]` by a test tagged
  `Validates: <ID>` or `unverified: <why>`. Future work lives in GitHub issues, not in docs.
- `docs/architecture.md` says how the code is built and why; `docs/adr/NNNN-title.md` records each
  decision (superseded, never deleted); `docs/lessons.md` holds rules learned and open problems,
  and is pruned when they are resolved.
- Update the docs and this file in the same PR as the code they describe.
- `README.md` is the user guide and `CHANGELOG.md` the user-visible history: a user-visible
  change adds its entry under the version it will release (the next patch after the highest
  tag, or `X.Y.0` when the PR changes `VERSION`).

## Increment lifecycle

1. **Start**: `script/stage start`. Anything red on main or in the last scheduled run, and any
   unanswered review comment on recent PRs, is fixed before new work. Read the open issues.
2. **Ready**: write the Definition of Ready in the PR body: the problem, requirement IDs or
   issue, acceptance criteria, what is out of scope.
3. **Test first**, then implement with the instrumentation flywheel.
4. **Verify** with the one verify command; it must be green before pushing.
5. **Docs** (requirements, architecture, ADRs, lessons, this file) in the same PR.
6. **Land**: PR, wait for `CI gate`, squash-merge, which releases (see Git, CI and PRs).
7. **Report**: after the merge, one PR comment with the Definition of Done checklist and the final
   increment report (below).

A plan that spans several increments lives in a GitHub tracking issue with a checklist, never in
a file. Never committed: per-PR report or readiness files, session state or resume files, ledgers
of merged PRs. That information belongs in the PR and its comments.

## Git, CI and PRs

- One branch per increment, cut from a fresh `origin/main`, named `feat/`, `fix/`, `docs/`,
  `chore/`, `refactor/` or `perf/` plus a short topic. Never reuse a branch after its PR merged;
  never name a branch for the tool that wrote it.
- Every change lands through a PR whose body fills `.github/pull_request_template.md`. CI checks
  the body, and that a PR titled `fix...` changes a test.
- `CI gate` is the one required check: a job that needs every other job. Wait for it
  (`script/stage pre-merge <PR>`), and fix every failure; never bypass it.
- Squash-merge as soon as `CI gate` is green, without asking: in a cloud session with
  `gh api -X PUT repos/<owner>/<repo>/pulls/<N>/merge -f merge_method=squash -f sha=<head sha>`
  (`script/stage post-merge <PR>` does this); locally with `gh pr merge --auto --squash`.
- Merging to main releases; nobody tags or publishes by hand.
- The cloud sessions' GitHub proxy refuses tag pushes, branch deletion and GraphQL: use the REST
  API (`gh api`), and leave branch deletion to GitHub's delete-on-merge and the branch-cleanup
  workflow.
- Commit messages: an imperative subject saying what changed; the body says why.
- Releases (`.github/workflows/release.yml`): CI succeeding for a push to main triggers it
  (`workflow_run`); it builds the commit CI tested. `script/release-version` picks the next
  patch after the highest tag of the series in `VERSION` (`.0` for a new series), refuses a
  version not higher than every tag and a commit that lacks the highest tag, and a commit already
  tagged finishes that release. The workflow creates the tag with `GITHUB_TOKEN`, builds static
  `gomediaimport-<version>-{darwin,linux}-{amd64,arm64}` binaries stamped through
  `-ldflags -X main.version`, writes `SHA256SUMS`, attests them with
  `actions/attest-build-provenance`, publishes with generated notes, and verifies the release
  with `gh attestation verify`. The `release` concurrency group serializes runs; scheduled and
  PR runs of CI never release; a dispatch re-runs the release for main's head.
- A minor or major series starts with a PR that edits `VERSION`; a major bump also changes the
  module path's `/vN` suffix (`TestModulePathMatchesVersionSeries` enforces it).
- CI (`.github/workflows/ci.yml`): `verify` (`script/verify` on ubuntu-latest and
  macos-latest), `process` (`script/process-check`, and `script/pr-check` on PRs), and
  `CI gate`; also weekly on main. Branch cleanup: `.github/workflows/branch-cleanup.yml`.

## Privacy review

Before every commit, read your own staged diff (`git diff --cached`) and remove anything private
about the maintainer, their family or their computers: people's names, addresses, personal email
addresses and accounts, hostnames, home-directory paths, IP addresses, tenant, client or vault
IDs, real personal documents or photos. Use neutral stand-ins in tests and examples. There is no
list of private terms to check against: judge the diff.

## Definition of Done

An increment is done when every line holds; the final report marks each ✅ or ❌ with evidence:

- [ ] Start stage clean: main CI, the last scheduled run and recent review comments were green or
      fixed first.
- [ ] Tests first; the regression test's red output before the fix is quoted.
- [ ] The verify command is green, and CI is green on the final commit.
- [ ] Traceability and doc checks (`script/process-check`, `script/pr-check`) are green.
- [ ] Docs and CLAUDE.md are updated in the same PR.
- [ ] Instrumentation reviewed on touched paths: what was added, what removed, or why it is
      adequate.
- [ ] Privacy review done on every commit's staged diff.
- [ ] Every review thread is answered or resolved.
- [ ] Merged; main CI green; the release it produced is linked.
- [ ] Clean up: only main and open PRs' branches remain on the remote; the working tree is clean.

### Final increment report

One PR comment after the merge, with these parts:

1. **Done / not done**: what the increment delivered, and anything left out.
2. **Definition of Ready**: as written, and whether it held.
3. **Requirements, ADRs and docs**: IDs added, changed or retired; ADRs; docs touched.
4. **Design decisions**: each with the alternatives considered and why they lost.
5. **Verification evidence**: commands, CI run links and durations, release link.
6. **Regression test**: name and red output before the fix (or "no bug fixed").
7. **Diagnosis**: for a bug, the root cause and how it was established.
8. **Instrumentation**: added, removed, and gaps still open.
9. **Performance**: before and after numbers, or "not touched".
10. **Privacy review**: done on every commit; anything removed.
11. **Smells**: what looked wrong but was out of scope.
12. **Follow-ups**: each filed as an issue, linked.
13. **Definition of Done**: the checklist above, each line ✅ or ❌ with evidence.
