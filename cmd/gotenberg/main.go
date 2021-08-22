package main

import (
	gotenbergapp "github.com/gotenberg/gotenberg/v7/internal/app/gotenberg"

	// Gotenberg modules.
	_ "github.com/gotenberg/gotenberg/v7/pkg/standard"
	// PDF engines.
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/pdfengine"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/pdfcpu"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/pdftk"
)

func main() {
	gotenbergapp.Run()
}
