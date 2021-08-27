#!/bin/bash

set -e

GOLANG_VERSION="$1"
GOTENBERG_VERSION="$2"
GOTENBERG_USER_GID="$3"
GOTENBERG_USER_UID="$4"
PDFTK_VERSION="$5"
DOCKER_REPOSITORY="$6"

GOTENBERG_VERSION="${GOTENBERG_VERSION//v}"
SEMVER=( ${GOTENBERG_VERSION//./ } )
VERSION_LENGTH=${#SEMVER[@]}

if [ $VERSION_LENGTH -ne 3 ]; then
    echo "$VERSION is not semver."
    exit 1
fi

docker buildx build \
  --build-arg GOLANG_VERSION="$GOLANG_VERSION" \
  --build-arg GOTENBERG_VERSION="$GOTENBERG_VERSION" \
  --build-arg GOTENBERG_USER_GID="$GOTENBERG_USER_GID" \
  --build-arg GOTENBERG_USER_UID="$GOTENBERG_USER_UID" \
  --build-arg PDFTK_VERSION="$PDFTK_VERSION" \
  --platform linux/amd64 \
  --platform linux/arm64 \
  -t "$DOCKER_REPOSITORY/gotenberg:latest" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}.${SEMVER[1]}" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}" \
  --push \
  -f build/Dockerfile .

# Cloud Run variant.
docker buildx build \
  --build-arg DOCKER_REPOSITORY="$DOCKER_REPOSITORY" \
  --build-arg GOTENBERG_VERSION="$GOTENBERG_VERSION" \
  --platform linux/amd64 \
  --platform linux/arm64 \
  -t "$DOCKER_REPOSITORY/gotenberg:latest-cloudrun" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}-cloudrun" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}.${SEMVER[1]}-cloudrun" \
  -t "$DOCKER_REPOSITORY/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}-cloudrun" \
  --push \
  -f build/Dockerfile.cloudrun .