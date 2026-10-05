#!/usr/bin/env sh
# Tests scripts/changed-paths.sh: feeds it a changed-file list and checks
# which CI groups come out true. Run by ci.yml's `changes` job before the
# classifier is trusted to skip anything.
set -eu

cd "$(dirname "$0")/.."

fail=0

# expect <description> <true-groups, space separated> <changed files...>
expect() {
  desc="$1"
  want="$2"
  shift 2

  got="$(printf '%s\n' "$@" | sh scripts/changed-paths.sh | sed -n 's/=true$//p' | sort | tr '\n' ' ' | sed 's/ $//')"
  want="$(printf '%s\n' $want | sort | tr '\n' ' ' | sed 's/ $//')"

  if [ "$got" != "$want" ]; then
    echo "FAIL: $desc" >&2
    echo "  want: $want" >&2
    echo "  got:  $got" >&2
    fail=1
  fi
}

expect "README only runs the prose and markdown checks" "markdown prose" README.md
expect "internal/ runs the Go, Docker, packaging and nix checks" "go docker_build release nix" internal/cli/cli.go
expect "a test file counts as Go" "go docker_build release nix" cmd/hush-hush-cli/get_test.go
expect "go.mod runs the Go checks" "go docker_build release nix" go.mod
expect "Dockerfile runs the Docker checks" "docker docker_build" Dockerfile
expect "Dockerfile.release feeds the packaging build too" "docker release" Dockerfile.release
expect "goreleaser config runs the packaging checks, and prettier reads it" "markdown release" .goreleaser.yml
expect "PKGBUILD runs only the pkgbuild check" "pkgbuild" PKGBUILD
expect "flake.lock runs only the nix check" "nix" flake.lock
expect "bun.lock runs only the markdown job" "markdown" bun.lock
expect "styles/ runs only the prose checks" "prose" styles/config/vocabularies/House/accept.txt
expect "a workflow other than ci.yml runs actionlint" "actionlint markdown" .github/workflows/release.yml
expect "ci.yml runs everything" "actionlint docker docker_build go markdown nix pkgbuild prose release" .github/workflows/ci.yml
expect "an unrelated file runs nothing" "" LICENSE
expect "no changed files runs nothing" ""

# An empty change list is only a real answer for a diff; the workflow
# passes --all for a push, where there is no base to diff against.
all="$(sh scripts/changed-paths.sh --all | grep -vc '=true' || true)"
[ "$all" = "0" ] || { echo "FAIL: --all left a group false" >&2; fail=1; }

exit "$fail"
