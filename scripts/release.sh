#!/bin/bash

set -e

DOCKER_REPO_GH="ghcr.io/onebrief"

GOLANG_VERSION="$1"
GOTENBERG_VERSION="$2"
DOCKER_REPOSIITORY="$3"

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
  --platform linux/amd64 \
  --platform linux/arm64 \
  --platform linux/386 \
  --platform linux/arm/v7 \
  -t "$DOCKER_REPO_GH/gotenberg:latest" \
  -t "$DOCKER_REPO_GH/gotenberg:${SEMVER[0]}" \
  -t "$DOCKER_REPO_GH/gotenberg:${SEMVER[0]}.${SEMVER[1]}" \
  -t "$DOCKER_REPO_GH/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}" \
  --push \
  -f build/Dockerfile.bc .

