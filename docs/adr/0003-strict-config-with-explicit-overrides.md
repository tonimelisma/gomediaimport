# 0003. Strict configuration, three-tier precedence and explicit boolean overrides

- Status: accepted
- Date: 2026-06-20

## Context and problem

Settings come from defaults, a YAML file and flags. A misspelled or removed key that is
silently ignored changes what an import deletes; a `--flag=false` that cannot beat a `true` in
the file makes the file impossible to override. Serves GMI-001 to GMI-005.

## Decision drivers

- A typo never silently changes behavior, especially around `delete_originals`.
- The command line always wins.
- Tests can run `run()` many times without shared state.

## Considered options

### Lenient YAML, flags merged by zero value

- Good: old config files keep loading.
- Bad: removed keys (`skip_thumbnails`, `checksum_imports`) are silently ignored; a false flag
  is indistinguishable from an absent one.
- Failure modes: an import does something other than the config says.

### Strict YAML, precedence by whether a flag was given

- Good: unknown keys fail loudly; `--flag=false` beats the file.
- Bad: removing a key breaks existing config files (a major version).
- Failure modes: a config error at start, before any file is touched.

## Decision

The config is parsed with `KnownFields`, so unknown, misspelled and removed keys (top level and
under removable volumes) are errors. Settings resolve defaults, then the file, then flags; a
boolean flag overrides the file whenever `wasFlagProvided(osArgs, flag)` sees it, in either
form. `run(osArgs)` takes its arguments as a parameter and the code keeps no global state.
`--quiet` forces verbose output off.

## Consequences

- Removing or renaming a key is a breaking change and needs a major version.
- Tests call `run()` with their own arguments and need no save and restore of globals.

## Confirmation

`TestParseConfigFile` (GMI-003), `TestWasFlagProvided` and `TestRunBooleanOverrideFalse`
(GMI-002), `TestSetDefaults` and `TestRunWorkersOverride` (GMI-001), the `TestValidateConfig*`
tests (GMI-005).

## Retirement proof

A config migration tool that rewrites old keys, making lenient parsing unnecessary anyway.

## Reversal plan

Drop `KnownFields` and warn on unknown keys instead; the precedence rule stays.
