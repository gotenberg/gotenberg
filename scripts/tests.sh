#!/bin/bash

set -e

DOCKER_REPOSITORY="$1"
CODE_COVERAGE="$2"

touch "$PWD/coverage.txt"
chmod 777 "$PWD/coverage.txt"
docker build -t "$DOCKER_REPOSITORY/gotenberg:tests" -f build/tests/Dockerfile .

if [ "$CODE_COVERAGE" = "1" ]; then
    docker run --rm -e "CODE_COVERAGE=$CODE_COVERAGE" -v "$PWD/coverage.txt:/gotenberg/tests/coverage.txt" "$DOCKER_REPOSITORY/gotenberg:tests"
else
    docker run --rm -e "CODE_COVERAGE=$CODE_COVERAGE" "$DOCKER_REPOSITORY/gotenberg:tests"
fi