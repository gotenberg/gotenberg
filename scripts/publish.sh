#!/bin/bash

set -e

GOLANG_VERSION=1.11.2
VERSION="$1"
DOCKER_USER="$2"
DOCKER_PASSWORD="$3"

docker login -u "$DOCKER_USER" -p "$DOCKER_PASSWORD"

SEMVER=( ${VERSION//./ } )   
VERSION_LENGTH=${#SEMVER[@]}

if [ $VERSION_LENGTH -ne 3 ]; then
    echo "$VERSION is not semver."
    exit 1
fi

docker build -t thecodingmachine/gotenberg:base -f build/base/Dockerfile .
docker build \
    --build-arg GOLANG_VERSION=${GOLANG_VERSION} \
    --build-arg VERSION=${VERSION} \
    -t thecodingmachine/gotenberg:${SEMVER[0]} \
    -t thecodingmachine/gotenberg:${SEMVER[0]}.${SEMVER[1]} \
    -t thecodingmachine/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]} \
    -f build/package/Dockerfile .

docker push "thecodingmachine/gotenberg:${SEMVER[0]}"
docker push "thecodingmachine/gotenberg:${SEMVER[0]}.${SEMVER[1]}"
docker push "thecodingmachine/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}"