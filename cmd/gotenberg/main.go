package main

import (
	"fmt"
	"runtime"
	gotenbergcmd "github.com/gotenberg/gotenberg/v7/cmd"

	// Gotenberg modules.
	_ "github.com/gotenberg/gotenberg/v7/pkg/standard"
)

func main() {
	fmt.Println(runtime.GOMAXPROCS(1))
	fmt.Println(runtime.GOMAXPROCS(1))
	gotenbergcmd.Run()
}
