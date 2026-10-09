#!/usr/bin/env sh
# Style: house voice, weasel words, corporate speak, the cliches proselint
# knows. Advice, not a gate - Vale only fails on error-severity alerts
# (MinAlertLevel in .vale.ini), which is why this script's own exit code is
# the real signal and nothing here downgrades it.
#
# With file arguments it lints only those and does not sync (the pre-commit
# hook passes the staged Markdown; run `vale sync` once on a fresh clone).
# With none it syncs the styles and lints the whole set, as pre-push and CI do.
set -eu

# renovate: datasource=docker depName=jdkato/vale
IMAGE="jdkato/vale:v3.17.1@sha256:7dba3c9104ba366f172d119022c4ec53a005f7d14dc1b80e285421a3f0b71657"

cd "$(dirname "$0")/.."

if [ "$#" -gt 0 ]; then
  SYNC=""
  FILES="$*"
else
  SYNC="vale sync && "
  FILES="README.md CONTRIBUTING.md CLAUDE.md SECURITY.md INSTALL.md"
fi

if command -v vale >/dev/null 2>&1; then
  [ -z "$SYNC" ] || vale sync
  # shellcheck disable=SC2086
  vale $FILES
else
  docker run --rm -v "$PWD:/work" -w /work --entrypoint sh "$IMAGE" \
    -c "${SYNC}vale $FILES"
fi
