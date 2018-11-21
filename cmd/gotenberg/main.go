package main

import (
	"fmt"
	"os"

	"github.com/thecodingmachine/gotenberg/internal/app/api"
	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
	"github.com/thecodingmachine/gotenberg/internal/pkg/process"
)

var version = "snapshot"

func main() {
	notify.Println(fmt.Sprintf("Gotenberg %s", version))
	if err := process.Start(); err != nil {
		notify.ErrPrintln(err)
		os.Exit(1)
	}
	if err := api.Start(); err != nil {
		notify.ErrPrintln(err)
		os.Exit(1)
	}
	os.Exit(0)
}
