package chromium

import (
	// Gotenberg modules (Chromium variant — no LibreOffice).
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/chromium"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/exiftool"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/pdfcpu"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/pdfengines"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/pdftk"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/prometheus"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/qpdf"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/webhook"
)
