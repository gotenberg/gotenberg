#!/bin/bash

set -e

GOLANG_VERSION="$1"
TINI_VERSION="$2"
DOCKER_REGISTRY="$3"
VERSION="$4"
DOCKER_USER="$5"
DOCKER_PASSWORD="$6"

docker login -u "$DOCKER_USER" -p "$DOCKER_PASSWORD"

VERSION="${VERSION//v}"
SEMVER=( ${VERSION//./ } )   
VERSION_LENGTH=${#SEMVER[@]}

if [ $VERSION_LENGTH -ne 3 ]; then
    echo "$VERSION is not semver."
    exit 1
fi

docker build \
    --build-arg VERSION=${VERSION} \
    --build-arg TINI_VERSION=${TINI_VERSION} \
    -t ${DOCKER_REGISTRY}/gotenberg:latest \
    -t ${DOCKER_REGISTRY}/gotenberg:${SEMVER[0]} \
    -t ${DOCKER_REGISTRY}/gotenberg:${SEMVER[0]}.${SEMVER[1]} \
    -t ${DOCKER_REGISTRY}/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]} \
    -f build/package/Dockerfile .

docker push "${DOCKER_REGISTRY}/gotenberg:latest"
docker push "${DOCKER_REGISTRY}/gotenberg:${SEMVER[0]}"
docker push "${DOCKER_REGISTRY}/gotenberg:${SEMVER[0]}.${SEMVER[1]}"
docker push "${DOCKER_REGISTRY}/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}"