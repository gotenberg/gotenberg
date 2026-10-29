package libreoffice

import (
	// Gotenberg modules (LibreOffice variant — no Chromium).
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/api"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/exiftool"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice/api"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice/pdfengine"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/pdf"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/pdfcpu"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/pdfengines"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/pdftk"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/prometheus"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/qpdf"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/webhook"
)
