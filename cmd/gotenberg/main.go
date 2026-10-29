package main

import (
	gotenbergcmd "github.com/gotenberg/gotenberg/v9/cmd"
	// Gotenberg modules.
	_ "github.com/gotenberg/gotenberg/v9/pkg/standard"
)

func main() {
	gotenbergcmd.Run()
}
