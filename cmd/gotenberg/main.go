package main

import (
	"os"

	"github.com/thecodingmachine/gotenberg/internal/app/cli"
	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
)

var version = "snapshot"

func main() {
	cli.SetVersion(version)
	if err := cli.Run(); err != nil {
		notify.ErrPrintln(err)
		os.Exit(1)
	}
	os.Exit(0)
}
