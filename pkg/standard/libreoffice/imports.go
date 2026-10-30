package libreoffice

import (
	// Gotenberg modules (LibreOffice variant — no Chromium).
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/api"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice/api"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/pdf"
	_ "github.com/gotenberg/gotenberg/v9/pkg/modules/webhook"
)
