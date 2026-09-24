package main

import (
	gotenbergcmd "github.com/gotenberg/gotenberg/v8/cmd"
	// Gotenberg modules (Chromium variant — no LibreOffice).
	_ "github.com/gotenberg/gotenberg/v8/pkg/standard/chromium"
)

func main() {
	gotenbergcmd.Run()
}
