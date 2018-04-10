#!/bin/bash

set -xe

# Statically checking Go source for errors and warnings.
gometalinter.v2 --disable-all -E vet -E gofmt -E misspell -E ineffassign -E goimports -E deadcode -E gocyclo --vendor ./...;

# Running tests according to current Gotenberg version.
if [[ "$VERSION" == "snapshot" ]]; then
    for d in $(go list ./... | grep -v vendor); do
        go test -race -cover $d;
    done
else
    echo "" > .ci/coverage.txt;
    for d in $(go list ./... | grep -v vendor); do
        go test -race -coverprofile=profile.out -covermode=atomic $d;
        if [ -f profile.out ]; then
            cat profile.out >> .ci/coverage.txt;
            rm profile.out;
        fi
    done
fi


# Builds the Linux binary.
if [ -f .ci/gotenberg ]; then
    rm .ci/gotenberg
fi

env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" && mv gotenberg .ci/;

# Bye!
exit 0;