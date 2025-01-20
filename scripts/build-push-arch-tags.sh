#!/bin/bash

set -e

GOLANG_VERSION="$1"
GOTENBERG_VERSION="$2"
GOTENBERG_USER_GID="$3"
GOTENBERG_USER_UID="$4"
NOTO_COLOR_EMOJI_VERSION="$5"
PDFTK_VERSION="$6"
PDFCPU_VERSION="$7"
DOCKER_REGISTRY="$8"
DOCKER_REPOSITORY="$9"
PLATFORM="${10}" # e.g., "linux/amd64" or "linux/386" or "linux/arm64" or "linux/arm/v7".

# Extract ARCH from PLATFORM (e.g. "linux/amd64" => "amd64"; "linux/arm64" => "arm64").
ARCH=$(echo "$PLATFORM" | cut -d/ -f2)
MAIN_TAG_SUFFIX="-$ARCH"

# Sanitize / check version (semver logic or fallback).
GOTENBERG_VERSION="${GOTENBERG_VERSION//v}" # Remove leading "v" if present
IFS='.' read -ra SEMVER <<< "$GOTENBERG_VERSION"
VERSION_LENGTH=${#SEMVER[@]}

TAGS=()
TAGS_CLOUD_RUN=()

if [ "$VERSION_LENGTH" -eq 3 ]; then
  MAJOR="${SEMVER[0]}"
  MINOR="${SEMVER[1]}"
  PATCH="${SEMVER[2]}"

  # Main tags with architecture suffix.
  TAGS+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:latest${MAIN_TAG_SUFFIX}")
  TAGS+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${MAJOR}${MAIN_TAG_SUFFIX}")
  TAGS+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${MAJOR}.${MINOR}${MAIN_TAG_SUFFIX}")
  TAGS+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${MAJOR}.${MINOR}.${PATCH}${MAIN_TAG_SUFFIX}")

  # If platform == linux/amd64, produce Cloud Run tags (no arch suffix).
  if [ "$PLATFORM" = "linux/amd64" ]; then
    TAGS_CLOUD_RUN+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:latest-cloudrun")
    TAGS_CLOUD_RUN+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${MAJOR}-cloudrun")
    TAGS_CLOUD_RUN+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${MAJOR}.${MINOR}-cloudrun")
    TAGS_CLOUD_RUN+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${MAJOR}.${MINOR}.${PATCH}-cloudrun")
  fi

else
  # Fallback for non-strict-semver versions.
  GOTENBERG_VERSION="${GOTENBERG_VERSION// /-}"
  GOTENBERG_VERSION="$(echo "$GOTENBERG_VERSION" | tr -cd '[:alnum:]._\-')"

  if [[ "$GOTENBERG_VERSION" =~ ^[\.\-] ]]; then
    GOTENBERG_VERSION="_${GOTENBERG_VERSION#?}"
  fi

  if [ "${#GOTENBERG_VERSION}" -gt 128 ]; then
    GOTENBERG_VERSION="${GOTENBERG_VERSION:0:128}"
  fi

  # Main tag with architecture suffix.
  TAGS+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${GOTENBERG_VERSION}${MAIN_TAG_SUFFIX}")

  # Cloud Run only if linux/amd64.
  if [ "$PLATFORM" = "linux/amd64" ]; then
    TAGS_CLOUD_RUN+=("-t" "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${GOTENBERG_VERSION}-cloudrun")
  fi
fi

# Build image.
echo "Building for platform: $PLATFORM (arch: $ARCH)"
echo "Using version: $GOTENBERG_VERSION"
echo "Main image tags (with arch suffix):"
for t in "${TAGS[@]}"; do
  if [ "$t" = "-t" ]; then
    continue
  fi
  echo " - $t"
done

docker buildx build \
  --build-arg GOLANG_VERSION="$GOLANG_VERSION" \
  --build-arg GOTENBERG_VERSION="$GOTENBERG_VERSION" \
  --build-arg GOTENBERG_USER_GID="$GOTENBERG_USER_GID" \
  --build-arg GOTENBERG_USER_UID="$GOTENBERG_USER_UID" \
  --build-arg NOTO_COLOR_EMOJI_VERSION="$NOTO_COLOR_EMOJI_VERSION" \
  --build-arg PDFTK_VERSION="$PDFTK_VERSION" \
  --build-arg PDFCPU_VERSION="$PDFCPU_VERSION" \
  --platform "$PLATFORM" \
  "${TAGS[@]}" \
  --push \
  -f build/Dockerfile \
  .

# Build Cloud Run variant if platform == linux/amd64.
if [ "$PLATFORM" = "linux/amd64" ]; then
  echo "Building Cloud Run variant for linux/amd64..."

  echo "Cloud Run tags (no arch suffix):"
  for t in "${TAGS_CLOUD_RUN[@]}"; do
    if [ "$t" = "-t" ]; then
      continue
    fi
    echo " - $t"
  done

  echo "Pulling $DOCKER_REGISTRY/$DOCKER_REPOSITORY:${GOTENBERG_VERSION}${MAIN_TAG_SUFFIX}"
  docker pull "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${GOTENBERG_VERSION}${MAIN_TAG_SUFFIX}"

  echo "Tagging $DOCKER_REGISTRY/$DOCKER_REPOSITORY:${GOTENBERG_VERSION}${MAIN_TAG_SUFFIX} into $DOCKER_REGISTRY/$DOCKER_REPOSITORY:$GOTENBERG_VERSION"
  docker image tag \
   "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:${GOTENBERG_VERSION}${MAIN_TAG_SUFFIX}" \
   "$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$GOTENBERG_VERSION"

  echo "Building Cloud Run variant using $DOCKER_REGISTRY/$DOCKER_REPOSITORY:$GOTENBERG_VERSION as base image"
  docker build \
    --build-arg DOCKER_REGISTRY="$DOCKER_REGISTRY" \
    --build-arg DOCKER_REPOSITORY="$DOCKER_REPOSITORY" \
    --build-arg GOTENBERG_VERSION="$GOTENBERG_VERSION" \
    "${TAGS_CLOUD_RUN[@]}" \
    --push \
    -f build/Dockerfile.cloudrun \
    .
else
  echo "Skipping Cloud Run variant (not linux/amd64)."
fi

# Output the actual tags for downstream usage (GitHub Actions, etc).
# We just parse out the '-t' references from TAGS + TAGS_CLOUD_RUN.
ALL_TAGS=()

extract_tags() {
  local arr=("$@")
  local extracted=()
  local i=0
  while [ $i -lt ${#arr[@]} ]; do
    if [ "${arr[$i]}" = "-t" ]; then
      i=$((i+1))
      extracted+=("${arr[$i]}")
    fi
    i=$((i+1))
  done
  echo "${extracted[@]}"
}

ARCH_TAGS_ARRAY=($(extract_tags "${TAGS[@]}"))
CLOUD_RUN_TAGS_ARRAY=($(extract_tags "${TAGS_CLOUD_RUN[@]}"))
ALL_TAGS=("${MAIN_TAGS_ARRAY[@]}" "${CLOUD_RUN_TAGS_ARRAY[@]}")

echo "All tags pushed:"
for ref in "${ALL_TAGS[@]}"; do
  echo " - $ref"
done

ARCH_TAGS_STR=$(IFS=","; echo "${ARCH_TAGS_ARRAY[*]}")

# Write them to the GitHub Actions output named "arch_tags".
echo "arch_tags=$ARCH_TAGS_STR" >> "$GITHUB_OUTPUT"
