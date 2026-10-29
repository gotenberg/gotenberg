#!/usr/bin/env bash
# Captures the version of a backing binary into a per-module file that the
# running Gotenberg process reads via gotenberg.BuildVersion, so it never spawns
# the binary just to report a version. This keeps cold start and the first
# request cheap, which matters on serverless platforms.
#
# Failure-tolerant by design: a probe that errors writes an empty file, and the
# runtime falls back to detecting the version live. A failing probe must never
# fail the image build.
#
# Usage: capture-version.sh <output-dir> <module-id> <bin> [args...]
set -u

dir="$1"
id="$2"
shift 2

mkdir -p "$dir"

# Run the probe once. On failure, keep going with empty output.
raw="$("$@" 2>/dev/null)" || raw=""

version="$(printf '%s\n' "$raw" | head -n1)"

printf '%s' "$version" | tr -d '\r' >"$dir/$id"
