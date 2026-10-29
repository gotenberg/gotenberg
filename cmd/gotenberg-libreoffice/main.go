package main

import (
	gotenbergcmd "github.com/gotenberg/gotenberg/v9/cmd"
	// Gotenberg modules (LibreOffice variant — no Chromium).
	_ "github.com/gotenberg/gotenberg/v9/pkg/standard/libreoffice"
)

func main() {
	gotenbergcmd.Run()
}
