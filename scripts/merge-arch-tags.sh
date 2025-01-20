#!/bin/bash

set -e

# The argument should be a comma-separated list of image tags.
# For example:
#   my-registry/my-repo:latest-amd64,my-registry/my-repo:latest-arm64,my-registry/my-repo:7.8.9-amd64,my-registry/my-repo:7.8.9-arm64
TAGS_INPUT="$1"
echo "Input tags: $TAGS_INPUT"

IFS=',' read -ra TAG_ARRAY <<< "$TAGS_INPUT"

#We gather tags into a map: baseTag -> list of arch-specific tags.
declare -A MERGE_MAP

# For each tag, remove any recognized arch suffix from the end:
# e.g.  :latest-amd64  ->  :latest
for TAG in "${TAG_ARRAY[@]}"; do
  BASE_TAG="$(echo "$TAG" \
    | sed -E 's/(-amd64|-arm64|-armv7|-386)$//')"

  # Put this arch-specific tag into the MERGE_MAP under the base tag.
  MERGE_MAP["$BASE_TAG"]+="$TAG "
done

# For each base tag, call "docker buildx imagetools create" to produce a multi-arch manifest.
# Example:
#  docker buildx imagetools create \
#    -t my-registry/my-repo:latest \
#    my-registry/my-repo:latest-amd64 \
#    my-registry/my-repo:latest-arm64
#
for BASE_TAG in "${!MERGE_MAP[@]}"; do
  ARCH_TAGS="${MERGE_MAP[$BASE_TAG]}"

  echo
  echo "Merging into base tag: $BASE_TAG"
  echo "Using arch-specific tags: $ARCH_TAGS"

  # Create the multi-arch manifest pointing to the base tag.
  docker buildx imagetools create -t "$BASE_TAG" $ARCH_TAGS
done

echo
echo "Done merging tags!"