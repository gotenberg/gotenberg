#!/bin/bash

set -e

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

docker push thecodingmachine/gotenberg:${SEMVER[0]}
docker push thecodingmachine/gotenberg:${SEMVER[0]}.${SEMVER[1]}
docker push thecodingmachine/gotenberg:${SEMVER[0]}.${SEMVER[1]}.${SEMVER[2]}