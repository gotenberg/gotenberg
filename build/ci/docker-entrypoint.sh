#!/bin/bash

set -e

# Statically checking Go source for errors and warnings.
golangci-lint run --tests=false --enable-all --disable=lll --disable=errcheck --disable=gosec --disable=gochecknoglobals --disable=gochecknoinits

# Running tests according to current Gotenberg version.
if [[ "$VERSION" == "snapshot" ]]; then
    for d in $(go list ./... | grep -v vendor); do
        go test -race -cover -covermode=atomic $d
    done
else
    if [ -f coverage.txt ]; then
        rm -f coverage.txt
    fi
    echo "" > coverage.txt
    for d in $(go list ./... | grep -v vendor); do
        go test -race -coverprofile=profile.out -covermode=atomic $d
        if [ -f profile.out ]; then
            cat profile.out >> coverage.txt
            rm -f profile.out
        fi
    done
fi