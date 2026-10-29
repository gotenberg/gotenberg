package main

import (
	gotenbergcmd "github.com/gotenberg/gotenberg/v9/cmd"
	// Gotenberg modules (Chromium variant — no LibreOffice).
	_ "github.com/gotenberg/gotenberg/v9/pkg/standard/chromium"
)

func main() {
	gotenbergcmd.Run()
}
