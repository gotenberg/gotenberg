package resource

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xassert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

// ArgKey is a type for
// arguments' keys.
type ArgKey string

const (
	// ResultFilenameArgKey is the key
	// of the argument "resultFilename".
	ResultFilenameArgKey ArgKey = "resultFilename"
	// WaitTimeoutArgKey is the key
	// of the argument "waitTimeout".
	WaitTimeoutArgKey ArgKey = "waitTimeout"
	// WebhookURLArgKey is the key
	// of the argument "webhookURL".
	WebhookURLArgKey ArgKey = "webhookURL"
	// WebhookURLTimeoutArgKey is the key
	// of the argument "webhookURLTimeout".
	WebhookURLTimeoutArgKey ArgKey = "webhookURLTimeout"
	// RemoteURLArgKey is the key
	// of the argument "remoteURL".
	RemoteURLArgKey ArgKey = "remoteURL"
	// WaitDelayArgKey is the key
	// of the argument "waitDelay".
	WaitDelayArgKey ArgKey = "waitDelay"
	// PaperWidthArgKey is the key
	// of the argument "paperWidth".
	PaperWidthArgKey ArgKey = "paperWidth"
	// PaperHeightArgKey is the key
	// of the argument "paperHeight".
	PaperHeightArgKey ArgKey = "paperHeight"
	// MarginTopArgKey is the key
	// of the argument "marginTop".
	MarginTopArgKey ArgKey = "marginTop"
	// MarginBottomArgKey is the key
	// of the argument "marginBottom".
	MarginBottomArgKey ArgKey = "marginBottom"
	// MarginLeftArgKey is the key
	// of the argument "marginLeft".
	MarginLeftArgKey ArgKey = "marginLeft"
	// MarginRightArgKey is the key
	// of the argument "marginRight".
	MarginRightArgKey ArgKey = "marginRight"
	// LandscapeArgKey is the key
	// of the argument "landscape".
	LandscapeArgKey ArgKey = "landscape"
)

/*
ArgKeys returns a slice
containing all available
arguments' keys.
*/
func ArgKeys() []ArgKey {
	return []ArgKey{
		ResultFilenameArgKey,
		WaitTimeoutArgKey,
		WebhookURLArgKey,
		WebhookURLTimeoutArgKey,
		RemoteURLArgKey,
		WaitDelayArgKey,
		PaperWidthArgKey,
		PaperHeightArgKey,
		MarginTopArgKey,
		MarginBottomArgKey,
		MarginLeftArgKey,
		MarginRightArgKey,
		LandscapeArgKey,
	}
}

/*
WaitTimeoutArg is a helper for retrieving
the "waitTimeout" argument as float64.

It also validates it against the application
configuration.
*/
func WaitTimeoutArg(r Resource, config conf.Config) (float64, error) {
	const op string = "resource.WaitTimeoutArg"
	result, err := r.Float64Arg(
		WaitTimeoutArgKey,
		config.DefaultWaitTimeout(),
		xassert.Float64NotInferiorTo(0),
		xassert.Float64NotSuperiorTo(config.MaximumWaitTimeout()),
	)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
WaitDelayArg is a helper for retrieving
the "waitDelay" argument as float64.

It also validates it against the application
configuration.
*/
func WaitDelayArg(r Resource, config conf.Config) (float64, error) {
	const (
		op               string  = "resource.WaitDelayArg"
		defaultWaitDelay float64 = 0.0
	)
	result, err := r.Float64Arg(
		WaitDelayArgKey,
		defaultWaitDelay,
		xassert.Float64NotInferiorTo(0.0),
		xassert.Float64NotSuperiorTo(config.MaximumWaitDelay()),
	)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
PaperSizeArgs is a helper for retrieving
the "paperWidth" and "paperHeight" arguments
as float64.
*/
func PaperSizeArgs(r Resource) (float64, float64, error) {
	const (
		op                 string  = "resource.PaperSizeArgs"
		defaultPaperWidth  float64 = 8.27
		defaultPaperHeight float64 = 11.7
	)
	resolver := func() (float64, float64, error) {
		paperWidth, err := r.Float64Arg(
			PaperWidthArgKey,
			defaultPaperWidth,
			xassert.Float64NotInferiorTo(0.0),
		)
		if err != nil {
			return defaultPaperWidth,
				defaultPaperHeight,
				err
		}
		paperHeight, err := r.Float64Arg(
			PaperHeightArgKey,
			defaultPaperHeight,
			xassert.Float64NotInferiorTo(0.0),
		)
		if err != nil {
			return defaultPaperWidth,
				defaultPaperHeight,
				err
		}
		return paperWidth,
			paperHeight,
			nil
	}
	paperWidth, paperHeight,
		err := resolver()
	if err != nil {
		return paperWidth,
			paperHeight,
			xerror.New(op, err)
	}
	return paperWidth,
		paperHeight,
		nil
}

/*
MarginArgs is a helper for retrieving
the "marginTop", "marginBottom", "marginLeft"
and "marginRight" arguments as float64.
*/
func MarginArgs(r Resource) (float64, float64, float64, float64, error) {
	const (
		op                  string  = "resource.MarginArgs"
		defaultMarginTop    float64 = 1.0
		defaultMarginBottom float64 = 1.0
		defaultMarginLeft   float64 = 1.0
		defaultMarginRight  float64 = 1.0
	)
	resolver := func() (float64, float64, float64, float64, error) {
		marginTop, err := r.Float64Arg(
			MarginTopArgKey,
			defaultMarginTop,
			xassert.Float64NotInferiorTo(0.0),
		)
		if err != nil {
			return defaultMarginTop,
				defaultMarginBottom,
				defaultMarginLeft,
				defaultMarginRight,
				err
		}
		marginBottom, err := r.Float64Arg(
			MarginBottomArgKey,
			defaultMarginBottom,
			xassert.Float64NotInferiorTo(0.0),
		)
		if err != nil {
			return defaultMarginTop,
				defaultMarginBottom,
				defaultMarginLeft,
				defaultMarginRight,
				err
		}
		marginLeft, err := r.Float64Arg(
			MarginLeftArgKey,
			defaultMarginLeft,
			xassert.Float64NotInferiorTo(0.0),
		)
		if err != nil {
			return defaultMarginTop,
				defaultMarginBottom,
				defaultMarginLeft,
				defaultMarginRight,
				err
		}
		marginRight, err := r.Float64Arg(
			MarginRightArgKey,
			defaultMarginRight,
			xassert.Float64NotInferiorTo(0.0),
		)
		if err != nil {
			return defaultMarginTop,
				defaultMarginBottom,
				defaultMarginLeft,
				defaultMarginRight,
				err
		}
		return marginTop,
			marginBottom,
			marginLeft,
			marginRight,
			nil
	}
	marginTop, marginBottom, marginLeft, marginRight,
		err := resolver()
	if err != nil {
		return marginTop,
			marginBottom,
			marginLeft,
			marginRight,
			xerror.New(op, err)
	}
	return marginTop,
		marginBottom,
		marginLeft,
		marginRight,
		nil
}
