package main

import (
	"fmt"
	"os"

	"github.com/thecodingmachine/gotenberg/internal/app/api"
	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
)

// version will be set on build time.
var version = "snapshot"

func main() {
	notify.Println(fmt.Sprintf("Gotenberg %s", version))
	if err := api.Start(); err != nil {
		notify.ErrPrintln(err)
		os.Exit(1)
	}
	os.Exit(0)
}
