#!/bin/sh
# Lays out site/reports/ from the test job's junit.xml and coverage.out
# (rules/published-reports.md). Runs on pull requests too, so a broken
# conversion fails before the merge; only the deploy is main-only.
set -eu

out="${1:-site/reports}"
commit="${GITHUB_SHA:-$(git rev-parse HEAD)}"
date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

mkdir -p "$out/tests" "$out/coverage"
cp junit.xml "$out/tests/junit.xml"
cp coverage.out "$out/coverage/coverage.out"
go tool gocover-cobertura <coverage.out >"$out/coverage/coverage.xml"
go tool cover -html=coverage.out -o "$out/coverage/index.html"

grep -q '<coverage ' "$out/coverage/coverage.xml"

cat >"$out/index.html" <<HTML
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>hush-hush-cli reports</title>
</head>
<body>
<h1>hush-hush-cli reports</h1>
<p>Commit <code>$commit</code>, built $date.</p>
<ul>
<li><a href="tests/junit.xml">Test results (JUnit XML)</a></li>
<li><a href="coverage/">Coverage (HTML)</a></li>
<li><a href="coverage/coverage.xml">Coverage (Cobertura XML)</a></li>
<li><a href="coverage/coverage.out">Coverage (Go native profile)</a></li>
</ul>
</body>
</html>
HTML
