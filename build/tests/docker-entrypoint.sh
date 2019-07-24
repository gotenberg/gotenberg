#!/bin/bash

set -xe

# Make sure the user running the
# tests is the Gotenberg user.
CURRENT_USER=$(whoami)
if [ "$CURRENT_USER" != "gotenberg" ]; then
    exit 1
fi

# Start the PM2 processes 
# (Google Chrome headless & unoconv listener).
go run github.com/thecodingmachine/gotenberg/test/cmd/pm2

# Run our tests.
go test -race -cover ./...

# Testing PM2 processes launch separatly for avoiding
# spending to much time on each tests depending on
# them.
#go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestChromeStart
#go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestUnoconvStart

# Running others tests.
#go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/pkg/config
#go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/pkg/random
#go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/pkg/standarderror
#go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/pkg/timeout
#go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/app/api

# Finally testing processes shutdown.
#go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestChromeShutdown
#go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestUnoconvShutdown