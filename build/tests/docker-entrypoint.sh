#!/bin/bash

set -xe

# Testing PM2 processes launch separatly for avoiding
# spending to much time on each tests depending on
# them.
go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestChromeStart
go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestUnoconvStart

# Running others tests.
go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/app/api
go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/internal/pkg/rand

# Finally testing processes shutdown.
go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestChromeShutdown
go test github.com/thecodingmachine/gotenberg/internal/pkg/pm2 -run TestUnoconvShutdown