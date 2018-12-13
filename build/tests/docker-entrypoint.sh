#!/bin/bash

set -xe

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

# Testing Go client.
go build -o /usr/local/bin/gotenberg cmd/gotenberg/main.go
gotenberg &
sleep 10
go test -race -cover -covermode=atomic github.com/thecodingmachine/gotenberg/pkg
sleep 5 # allows Gotenberg to remove generated files (concurrent requests).