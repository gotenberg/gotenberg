#!/bin/bash

set -e

DOCKER_REGISTRY="$1"
CODE_COVERAGE="$2"

touch "$PWD/coverage.txt"
chmod 777 "$PWD/coverage.txt"
docker build -t "$DOCKER_REGISTRY/gotenberg:tests" -f build/tests/Dockerfile .

if [ "$CODE_COVERAGE" = "1" ]; then
    docker run --rm -e "CODE_COVERAGE=$CODE_COVERAGE" -v "$PWD/coverage.txt:/gotenberg/tests/coverage.txt" "$DOCKER_REGISTRY/gotenberg:tests"
else
    docker run --rm -e "CODE_COVERAGE=$CODE_COVERAGE" "$DOCKER_REGISTRY/gotenberg:tests"
fi