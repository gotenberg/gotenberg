package standard

import (
	// Standard Gotenberg modules.
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/chromium"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/gc"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/pdfengine"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/unoconv"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/logging"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/pdfcpu"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/pdfengines"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/pdftk"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/prometheus"
	_ "github.com/gotenberg/gotenberg/v7/pkg/modules/webhook"
)
