#!/bin/bash

# Exit early.
# See: https://www.gnu.org/savannah-checkouts/gnu/bash/manual/bash.html#The-Set-Builtin.
set -e

# Source dot env file.
source .env

# Arguments.
tags=""
alternate_registry=""
dry_run=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --tags)
      tags="$2"
      shift 2
      ;;
    --alternate-registry)
      alternate_registry="$2"
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

echo "Merge tag(s) 👷"
echo

echo "Tag(s) to merge:"
IFS=',' read -ra tags_to_merge <<< "$tags"
for tag in "${tags_to_merge[@]}"; do
  echo "- $tag"
done

if [ -n "$alternate_registry" ]; then
  echo "⚠️ Will also push to $alternate_registry registry"
fi

if [ "$dry_run" = "true" ]; then
  echo "🚧 Dry run"
fi
echo

# Build merge map.
declare -A merge_map

for tag in "${tags_to_merge[@]}"; do
  target_tag="${tag//-amd64/}"
  target_tag="${target_tag//-arm64/}"
  target_tag="${target_tag//-arm/}"
  target_tag="${target_tag//-386/}"

  merge_map["$target_tag"]+="$tag "
done

# Merge tags.
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

for target in "${!merge_map[@]}"; do
  IFS=' ' read -ra source_tags <<< "${merge_map[$target]}"

  cmd="docker buildx imagetools create \
       -t $target \
       ${source_tags[*]}
   "
  run_cmd "$cmd"

  echo "➡️ $target pushed"
  echo
  if [ -n "$alternate_registry" ]; then
    alternate_target="${target/$DOCKER_REGISTRY/$alternate_registry}"
    cmd="docker buildx imagetools create \
            -t $alternate_target \
            $target
        "
    run_cmd "$cmd"

    echo "➡️ $alternate_target pushed"
    echo
  fi
done

echo "✅ Done!"
exit 0
