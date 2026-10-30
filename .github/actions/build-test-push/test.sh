#!/bin/bash

# Exit early.
# See: https://www.gnu.org/savannah-checkouts/gnu/bash/manual/bash.html#The-Set-Builtin.
set -e

# Source dot env file.
source .env

# Arguments.
version=""
platform=""
alternate_repository=""
dry_run=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --version)
      version="${2//v/}"
      shift 2
      ;;
    --platform)
      platform="$2"
      shift 2
      ;;
    --alternate-repository)
      alternate_repository="$2"
      shift 2
      ;;
    --dry-run)
      dry_run="$2"
      shift 2
      ;;
    *)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

echo "Integration testing 🧪"
echo

echo "Gotenberg version: $version"
echo "Target platform: $platform"

repository=$DOCKER_REPOSITORY

if [ -n "$alternate_repository" ]; then
  echo "⚠️ Using $alternate_repository for DOCKER_REPOSITORY"
  repository=$alternate_repository
fi

if [ "$dry_run" = "true" ]; then
  echo "🚧 Dry run"
fi
echo

# Test image.
run_cmd() {
  local cmd="$1"

  if [ "$dry_run" = "true" ]; then
    echo "🚧 Dry run - would run the following command:"
    echo "$cmd"
    echo
  else
    echo "⚙️ Running command:"
    echo "$cmd"
    eval "$cmd"
    echo
  fi
}

cmd="make test-integration DOCKER_REPOSITORY=$repository GOTENBERG_VERSION=$version PLATFORM=$platform CONCURRENCY=3"
run_cmd "$cmd"

# The variants carry one engine each: run the scenarios of that engine.
cmd="make test-integration DOCKER_REPOSITORY=$repository GOTENBERG_VERSION=$version PLATFORM=$platform CONCURRENCY=3 VARIANT=chromium TAGS=chromium"
run_cmd "$cmd"

cmd="make test-integration DOCKER_REPOSITORY=$repository GOTENBERG_VERSION=$version PLATFORM=$platform CONCURRENCY=3 VARIANT=libreoffice TAGS=libreoffice"
run_cmd "$cmd"

echo "✅ Done!"
exit 0
