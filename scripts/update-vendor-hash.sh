#!/usr/bin/env sh
# Rewrites flake.nix's vendorHash to whatever Nix says it should be.
#
# vendorHash is a fixed-output hash of the vendored Go modules, so any
# go.mod/go.sum change invalidates it, and Dependabot can't edit Nix files.
# Nix only reveals the right value by failing a build and printing it
# ("got: sha256-..."), so this builds, reads that line, and writes it back.
#
# Exits 0 when the hash was already right or has been fixed. Exits non-zero
# for any other build failure: a broken derivation is a real red, not
# something to paper over with a hash edit.
set -eu

nix="${NIX:-nix}"
file="${FLAKE_FILE:-flake.nix}"

log="$(mktemp)"
trap 'rm -f "$log"' EXIT

if "$nix" build .#default --no-link -L >"$log" 2>&1; then
  echo "vendorHash already correct"
  exit 0
fi

new="$(sed -n 's/^.*got: *\(sha256-[A-Za-z0-9+\/]*=*\).*$/\1/p' "$log" | head -n 1)"
if [ -z "$new" ]; then
  echo "nix build failed, and not on a hash mismatch:" >&2
  tail -n 40 "$log" >&2
  exit 1
fi

sed -i "s|^\( *vendorHash = \"\)sha256-[^\"]*\(\";\)|\1$new\2|" "$file"
if ! grep -q "vendorHash = \"$new\";" "$file"; then
  echo "could not rewrite vendorHash in $file" >&2
  exit 1
fi
echo "vendorHash updated to $new"
