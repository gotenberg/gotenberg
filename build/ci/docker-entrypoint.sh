#!/bin/bash

set -e

# Statically checking Go source for errors and warnings.
golangci-lint run -D errcheck -E gofmt -E golint -E misspell -E prealloc -E gocyclo -E goconst

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

# Removing previous Gotenberg binary if it exists.
if [ -f build/package/gotenberg ]; then
    rm -f build/package/gotenberg
fi

# Building Gotenberg binary.
env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/package/gotenberg -ldflags "-X main.version=${VERSION}" cmd/gotenberg/main.go