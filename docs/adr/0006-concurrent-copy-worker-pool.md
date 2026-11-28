# 0006. Copies run in a worker pool; enumeration stays sequential

- Status: accepted
- Date: 2025-02-08

## Context and problem

Copying from a card to an SSD or network disk is bound by I/O latency; one copy at a time leaves
throughput unused. Enumeration and planning have to see every file in order to name and
de-duplicate them deterministically. Serves GMI-040 and GMI-045.

## Decision drivers

- Faster imports from fast media.
- Deterministic destination names regardless of copy order.
- Progress that stays readable.

## Considered options

### Sequential copy

- Good: simplest.
- Bad: slow on fast media.
- Failure modes: none beyond speed.

### Parallel enumeration and copy

- Good: maximum overlap.
- Bad: names and duplicates depend on timing.
- Failure modes: non-deterministic names.

### Sequential enumeration and planning, a pool for copying

- Good: names are fixed before any copy starts; copies overlap.
- Bad: a mutex around shared progress state.
- Failure modes: contention if the pool is too large for the media.

## Decision

Enumerate and plan sequentially, then copy with `--workers` goroutines (default 4), scheduling
jobs size-interleaved for balanced load. `progressTracker` owns the progress display (atomic size
tracking, a sticky ANSI line on a terminal).

## Consequences

- `workers` is a user-tunable setting; 0 means the default.
- Copy errors are collected across workers (see ADR 0004).

## Confirmation

`TestCopyFilesConcurrent`, `TestEffectiveWorkers`, `TestRunWorkersOverride` (GMI-040).

## Retirement proof

Measurements showing one worker is as fast on the media people use.

## Reversal plan

Set the default to 1; the pool code can stay.
