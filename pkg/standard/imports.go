package standard

import (
	// Standard Gotenberg modules.
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/chromium"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/pdfengine"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/logging"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/pdfcpu"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/pdfengines"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/pdftk"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/prometheus"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/qpdf"
	_ "github.com/gotenberg/gotenberg/v8/pkg/modules/webhook"
)
