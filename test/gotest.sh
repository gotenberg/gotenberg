#!/bin/bash

set -x

# TODO: remove -buildvcs=false when fix for https://github.com/golang/go/issues/51723 is live.
go test -buildvcs=false -race -covermode=atomic -coverprofile=/tests/coverage.txt ./...
go tool cover -html=coverage.txt -o /tests/coverage.html