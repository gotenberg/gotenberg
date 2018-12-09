#!/bin/bash

set -xe

# Statically checking Go source for errors and warnings.
golangci-lint run --tests=false --enable-all --disable=lll --disable=errcheck --disable=gosec --disable=gochecknoglobals --disable=gochecknoinits

# Testing PM2 processes launch separatly for avoiding
# spending to much time on each tests depending on
# them.
go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestChromeLaunch
go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestUnoconvLaunch

# Running others tests.
go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/app/api
go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/pkg/printer
go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/pkg/rand

# Finally testing processes shutdown.
go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestChromeShutdown
go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestUnoconvShutdown

# Running tests according to current Gotenberg version.
#if [[ "$VERSION" == "snapshot" ]]; then
#    for d in $(go list ./... | grep -v vendor); do
#        go test -race -cover -covermode=atomic $d
#    done
#else
#    if [ -f coverage.txt ]; then
#        rm -f coverage.txt
#    fi
#    echo "" > coverage.txt
#    for d in $(go list ./... | grep -v vendor); do
#        go test -race -coverprofile=profile.out -covermode=atomic $d
#        if [ -f profile.out ]; then
#            cat profile.out >> coverage.txt
#            rm -f profile.out
#        fi
#    done
#fi