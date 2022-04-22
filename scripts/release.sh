#!/bin/bash

set -e

DOCKER_REPO_GH="ghcr.io/onebrief"
DOCKER_REPO_HEROKU="registry.heroku.com/bc-gotenberg/web"

GOLANG_VERSION="$1"
GOTENBERG_VERSION="$2"
GOTENBERG_USER_GID="$3"
GOTENBERG_USER_UID="$4"
NOTO_COLOR_EMOJI_VERSION="$5"
PDFTK_VERSION="$6"
DOCKER_REPOSITORY="$7"

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
  --build-arg NOTO_COLOR_EMOJI_VERSION="$NOTO_COLOR_EMOJI_VERSION" \
  --build-arg PDFTK_VERSION="$PDFTK_VERSION" \
  --platform linux/amd64 \
  -t "$DOCKER_REPO_GH/gotenberg:latest" \
  -t "$DOCKER_REPO_GH/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}" \
  -t "$DOCKER_REPO_HEROKU" \
  --push \
  -f build/Dockerfile.bc .

# release app on heroku
heroku container:release web --app bc-gotenberg