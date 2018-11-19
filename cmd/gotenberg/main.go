package main

import (
	"fmt"
	"os"

	"github.com/thecodingmachine/gotenberg/internal/app/cli"
	"gopkg.in/alecthomas/kingpin.v2"
)

var version = "snapshot"

func main() {
	cli.SetVersion(version)
	if err := cli.Run(); err != nil {
		kingpin.Errorf("%v", err)
		os.Exit(1)
	}
	fmt.Println("Bye!")
	os.Exit(0)
}
