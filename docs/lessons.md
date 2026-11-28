# Lessons

Bounded: rules learned the hard way that are not obvious from the code, and problems still
open. An entry is deleted when its problem is solved or its rule moves into a test, an ADR or
CLAUDE.md.

## Rules learned

- Pin every tool CI installs. `govulncheck@latest` started requiring a newer Go than CI ran, and
  main stayed red at the install step until someone looked; a pinned version changes only through
  a reviewed bump.
- A check must fail on the problem it guards. `test -z "$(gofmt -l ./...)"` passed for any code,
  because gofmt reads `./...` as a missing file, prints the error to stderr and nothing to stdout.
  `gofmt -l .` is the working form.
- A Go module at v2 or later needs the `/vN` suffix in its module path, or Go ignores those tags:
  `go install ...@latest` kept installing v1.9.0 through the v2 and v3 releases.

## Open problems

- macOS volume labels are read with a raw `getattrlist` syscall that x/sys marks deprecated
  (#11).
