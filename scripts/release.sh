#!/bin/bash

set -e

GOLANG_VERSION="$1"
GOTENBERG_VERSION="$2"
GOTENBERG_USER_GID="$3"
GOTENBERG_USER_UID="$4"
NOTO_COLOR_EMOJI_VERSION="$5"
PDFTK_VERSION="$6"
DOCKER_REPOSITORY="$7"

if [ "$GOTENBERG_VERSION" == "edge" ]; then
  docker buildx build \
    --build-arg GOLANG_VERSION="$GOLANG_VERSION" \
    --build-arg GOTENBERG_VERSION="$GOTENBERG_VERSION" \
    --build-arg GOTENBERG_USER_GID="$GOTENBERG_USER_GID" \
    --build-arg GOTENBERG_USER_UID="$GOTENBERG_USER_UID" \
    --build-arg NOTO_COLOR_EMOJI_VERSION="$NOTO_COLOR_EMOJI_VERSION" \
    --build-arg PDFTK_VERSION="$PDFTK_VERSION" \
    --platform linux/amd64 \
    --platform linux/arm64 \
    --platform linux/386 \
    --platform linux/arm/v7 \
    -t "$DOCKER_REPOSITORY/gotenberg:edge" \
    --push \
    -f build/Dockerfile .

  # Cloud Run variant.
  # Only linux/amd64! See https://github.com/gotenberg/gotenberg/issues/505#issuecomment-1264679278.
  docker buildx build \
    --build-arg DOCKER_REPOSITORY="$DOCKER_REPOSITORY" \
    --build-arg GOTENBERG_VERSION="$GOTENBERG_VERSION" \
    --platform linux/amd64 \
    -t "$DOCKER_REPOSITORY/gotenberg:edge-cloudrun" \
    --push \
    -f build/Dockerfile.cloudrun .

  exit 0
fi

GOTENBERG_VERSION="${GOTENBERG_VERSION//v}"
IFS='.' read -ra SEMVER <<< "$GOTENBERG_VERSION"
VERSION_LENGTH=${#SEMVER[@]}

if [ "$VERSION_LENGTH" -ne 3 ]; then
  echo "$VERSION is not semver."
  exit 1
fi

docker buildx build \
  --build-arg GOLANG_VERSION="$GOLANG_VERSION" \
  --build-arg GOTENBERG_VERSION="$GOTENBERG_VERSION" \
  --build-arg GOTENBERG_USER_GID="$GOTENBERG_USER_GID" \
  --build-arg GOTENBERG_USER_UID="$GOTENBERG_USER_UID" \
  --build-arg NOTO_COLOR_EMOJI_VERSION="$NOTO_COLOR_EMOJI_VERSION" \
  --build-arg PDFTK_VERSION="$PDFTK_VERSION" \
  --platform linux/amd64 \
  --platform linux/arm64 \
  --platform linux/386 \
  --platform linux/arm/v7 \
  -t "$DOCKER_REPOSITORY/gotenberg:latest" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}.${SEMVER[1]}" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}" \
  --push \
  -f build/Dockerfile .

# Cloud Run variant.
# Only linux/amd64! See https://github.com/gotenberg/gotenberg/issues/505#issuecomment-1264679278.
docker buildx build \
  --build-arg DOCKER_REPOSITORY="$DOCKER_REPOSITORY" \
  --build-arg GOTENBERG_VERSION="$GOTENBERG_VERSION" \
  --platform linux/amd64 \
  -t "$DOCKER_REPOSITORY/gotenberg:latest-cloudrun" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}-cloudrun" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}.${SEMVER[1]}-cloudrun" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}-cloudrun" \
  --push \
  -f build/Dockerfile.cloudrun .