#!/bin/bash

# Exit early.
# See: https://www.gnu.org/savannah-checkouts/gnu/bash/manual/bash.html#The-Set-Builtin.
set -e

# Source dot env file.
source .env

# Arguments.
tags=""
dry_run=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --tags)
      tags="$2"
      shift 2
      ;;
    --dry-run)
      dry_run=$2
      shift 2
      ;;
    *)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

echo "Push tag(s) 📦"
echo

echo "Tag(s) to push:"
IFS=',' read -ra tags_to_push <<< "$tags"
for tag in "${tags_to_push[@]}"; do
  echo "- $tag"
done

if [ "$dry_run" = "true" ]; then
  echo "🚧 Dry run"
fi
echo

# Push tags.
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

for tag in "${tags_to_push[@]}"; do
  cmd="docker push $tag"
  run_cmd "$cmd"

  echo "➡️ $tag pushed"
  echo
done

echo "✅ Done!"
exit 0
