package main

import (
	gotenbergcmd "github.com/gotenberg/gotenberg/v8/cmd"
	// Gotenberg modules (LibreOffice variant — no Chromium).
	_ "github.com/gotenberg/gotenberg/v8/pkg/standard/libreoffice"
)

func main() {
	gotenbergcmd.Run()
}
