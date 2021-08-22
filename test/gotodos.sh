#!/bin/bash

set -x

golangci-lint run \
	--no-config \
	--disable-all \
	--enable godox