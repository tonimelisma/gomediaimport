#!/bin/sh
# Tests for script/release-version against throwaway git repositories.
# Validates: GMI-081
# Run: script/release-version_test.sh
set -u
root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
fails=0
ok() { printf 'ok   %s\n' "$1"; }
bad() { printf 'FAIL %s\n' "$1"; fails=$((fails + 1)); }

# A fresh repository whose first commit has VERSION=$1.
repo() {
  rm -rf "$work/r" && mkdir "$work/r" && cd "$work/r" || exit 1
  git init -q && git config user.email t@example.com && git config user.name t
  commit "$1"
}
commit() { printf '%s\n' "$1" >VERSION && git add VERSION && git commit -q --allow-empty -m "c $1"; }
expect() { # name, expected stdout (or FAIL)
  if out=$("$root/script/release-version" HEAD 2>/dev/null); then :; else out=FAIL; fi
  if [ "$out" = "$2" ]; then ok "$1"; else bad "$1: got '$out', want '$2'"; fi
}

repo 3.0; git tag v3.0.1 && commit 3.0
expect "next patch in the series" "$(printf 'version=v3.0.2\ncreate_tag=true\nlatest=true')"

commit 3.1
expect "new minor series starts at .0" "$(printf 'version=v3.1.0\ncreate_tag=true\nlatest=true')"

commit 4.0
expect "new major series starts at .0" "$(printf 'version=v4.0.0\ncreate_tag=true\nlatest=true')"

repo 3.0; git tag v3.0.1 && commit 2.9
expect "a series lower than the highest tag is refused" FAIL

repo 3.0; git tag v3.0.1
expect "a tagged commit finishes its release" "$(printf 'version=v3.0.1\ncreate_tag=false\nlatest=true')"

repo 3.0; old=$(git rev-parse HEAD); commit 3.0; git tag v3.0.5; git checkout -q "$old"
expect "a commit without the highest tag is refused" FAIL

repo 3; expect "a VERSION that is not MAJOR.MINOR is refused" FAIL
repo 03.0; expect "a leading zero is refused" FAIL

repo 1.2; expect "no tags yet starts the series at .0" "$(printf 'version=v1.2.0\ncreate_tag=true\nlatest=true')"

repo 3.0; git tag v3.0.9; commit 3.0; git tag v3.0.10 && commit 3.0
expect "patches sort numerically" "$(printf 'version=v3.0.11\ncreate_tag=true\nlatest=true')"

if [ "$fails" -ne 0 ]; then echo "$fails failed"; exit 1; fi
echo "all passed"
