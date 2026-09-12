#!/usr/bin/env bash
# Publishes/updates the hush-hush-cli-bin AUR package for one release, and
# leaves this repo's own PKGBUILD bumped to match.
#
# Run from the repo root with VERSION set (the tag without its leading
# "v", e.g. "1.4.7") and an SSH key for the AUR's `aur` user already
# loaded (CI: the AUR_SSH_KEY secret).
#
# The AUR repo is a separate git history from this one -- it's cloned
# fresh into a scratch directory each run, not kept as a subtree here.
set -euo pipefail

: "${VERSION:?VERSION must be set, e.g. VERSION=1.4.7}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PKGBUILD="$REPO_ROOT/PKGBUILD"

echo "==> Bumping pkgver to $VERSION"
sed -i "s/^pkgver=.*/pkgver=$VERSION/" "$PKGBUILD"
sed -i "s/^pkgrel=.*/pkgrel=1/" "$PKGBUILD"

echo "==> Recomputing checksums against the real release assets, per architecture"
# Deliberately not `updpkgsums PKGBUILD`: rules/pkgbuild.md documents that
# it resolves $CARCH once, from the machine actually running it, so a
# single-arch CI runner can silently produce a wrong checksum for every
# declared arch but its own. That trap needs source_<arch>'s own URL to
# reference $CARCH to bite -- this PKGBUILD's URLs don't -- but hashing
# each arch's real source directly sidesteps the whole class of bug
# instead of relying on that staying true.
source "$PKGBUILD"
for target_arch in "${arch[@]}"; do
  declare -n sources="source_${target_arch}"
  url="${sources[0]#*::}"
  sum="$(curl -fsSL "$url" | sha256sum | cut -d' ' -f1)"
  sed -i "s|^sha256sums_${target_arch}=.*|sha256sums_${target_arch}=('$sum')|" "$PKGBUILD"
done

echo "==> Cloning the AUR package repo"
SCRATCH="$(mktemp -d)"
trap 'rm -rf "$SCRATCH"' EXIT
git clone ssh://aur@aur.archlinux.org/hush-hush-cli-bin.git "$SCRATCH/aur"
cp "$PKGBUILD" "$SCRATCH/aur/PKGBUILD"

cd "$SCRATCH/aur"

echo "==> Regenerating .SRCINFO"
makepkg --printsrcinfo > .SRCINFO

# `git diff` (unstaged) never shows an untracked file as different, which
# a brand first publish always is -- `git add` then check the index
# against HEAD instead, which handles "nothing committed yet" the same
# way as "already published, nothing changed."
git add PKGBUILD .SRCINFO
if git diff --cached --quiet; then
  echo "No change against the published package; nothing to push."
  exit 0
fi

git -c user.name="hush-hush-cli release" -c user.email="ryan@andthensome.nl" \
  commit -m "release $VERSION"

echo "==> Pushing to the AUR"
git push origin master || {
  echo "::error::Failed to push hush-hush-cli-bin $VERSION to the AUR" >&2
  exit 1
}
