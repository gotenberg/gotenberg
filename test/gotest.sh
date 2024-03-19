#!/bin/bash

set -x

go test -race -covermode=atomic -coverprofile=/tests/coverage.txt ./...
RESULT=$?

go tool cover -html=coverage.txt -o /tests/coverage.html

exit $RESULT