#!/usr/bin/env sh
# Works out which CI check groups a change touches (rules/ci.md: run a
# check only when the files it covers change). Reads changed file names on
# stdin, one per line, and prints `<group>=true|false` for each group, the
# format $GITHUB_OUTPUT takes. `--all` skips stdin and marks every group
# true, for a push to main where there is no base to diff against.
#
# Each group lists the files it covers, the tool's own config and
# lockfile, and ci.yml itself, so editing the pipeline runs everything.
# Always-on jobs (commits, secrets, pr-title) don't appear here. When you
# add a job to ci.yml, give it a group here and a case in
# changed-paths.test.sh, and mirror the glob in lefthook.yml.
set -eu

# Every group also fires on a change to the pipeline itself.
PIPELINE='^\.github/workflows/ci\.yml$|^scripts/changed-paths(\.test)?\.sh$'

# Go source, module files, linter config, and anything the Go jobs read.
GO='\.go$|^go\.(mod|sum)$|^\.golangci\.yml$'

changed="$(mktemp)"
trap 'rm -f "$changed"' EXIT

if [ "${1:-}" = "--all" ]; then
  echo .github/workflows/ci.yml >"$changed"
else
  cat >"$changed"
fi

# group <name> <extended regex>: true when any changed file matches it or
# the pipeline pattern.
group() {
  if grep -Eq -e "$2" -e "$PIPELINE" "$changed"; then
    echo "$1=true"
  else
    echo "$1=false"
  fi
}

group go "$GO"
group actionlint '^\.github/workflows/'
group docker '^(Dockerfile|Dockerfile\.release|\.dockerignore|\.hadolint\.yaml)$'
# The from-source Dockerfile copies the whole tree, so Go changes can break it.
group docker_build "^(Dockerfile|\.dockerignore)$|$GO"
# goreleaser builds the binary, the man pages and the Docker image.
group release "^(\.goreleaser\.ya?ml|Dockerfile\.release)$|^manpages/|$GO"
group pkgbuild '^(PKGBUILD|\.SRCINFO)$|\.install$'
# The flake builds from source, so Go changes can stale its vendorHash.
group nix "^flake\.(nix|lock)$|$GO"
# prettier covers every md/yml/yaml file; bun's own files and the
# markdown/prettier config round it out.
group markdown '\.(md|ya?ml)$|^(package\.json|bun\.lock|\.prettierrc\.json|\.prettierignore|\.markdownlint-cli2\.yaml|\.editorconfig)$'
group prose '\.md$|^styles/|^\.vale\.ini$|^\.ltex\.json$|^scripts/lint-(vale|ltex)\.sh$'
