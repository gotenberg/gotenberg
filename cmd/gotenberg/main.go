package main

import (
	gotenbergcmd "github.com/gotenberg/gotenberg/v7/cmd"

	// Gotenberg modules.
	_ "github.com/gotenberg/gotenberg/v7/pkg/standard"
)

func main() {
	gotenbergcmd.Run()
}
