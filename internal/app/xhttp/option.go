package xhttp

import (
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

func mergePrinterOptions(r resource.Resource, config conf.Config) (printer.MergePrinterOptions, error) {
	const op string = "xhttp.mergePrinterOptions"
	waitTimeout, err := resource.WaitTimeoutArg(r, config)
	if err != nil {
		return printer.MergePrinterOptions{}, xerror.New(op, err)
	}
	return printer.MergePrinterOptions{
		WaitTimeout: waitTimeout,
	}, nil
}

func chromePrinterOptions(r resource.Resource, config conf.Config) (printer.ChromePrinterOptions, error) {
	const op string = "xhttp.chromePrinterOptions"
	resolver := func() (printer.ChromePrinterOptions, error) {
		waitTimeout, err := resource.WaitTimeoutArg(r, config)
		if err != nil {
			return printer.ChromePrinterOptions{}, err
		}
		waitDelay, err := resource.WaitDelayArg(r, config)
		if err != nil {
			return printer.ChromePrinterOptions{}, err
		}
		headerHTML, footerHTML,
			err := resource.HeaderFooterContents(r, config)
		if err != nil {
			return printer.ChromePrinterOptions{}, err
		}
		paperWidth, paperHeight,
			err := resource.PaperSizeArgs(r, config)
		if err != nil {
			return printer.ChromePrinterOptions{}, err
		}
		marginTop, marginBottom, marginLeft, marginRight,
			err := resource.MarginArgs(r, config)
		if err != nil {
			return printer.ChromePrinterOptions{}, err
		}
		landscape, err := r.BoolArg(resource.LandscapeArgKey, false)
		if err != nil {
			return printer.ChromePrinterOptions{}, err
		}
		googleChromeRpccBufferSize, err := resource.GoogleChromeRpccBufferSizeArg(r, config)
		if err != nil {
			return printer.ChromePrinterOptions{}, err
		}
		return printer.ChromePrinterOptions{
			WaitTimeout:    waitTimeout,
			WaitDelay:      waitDelay,
			HeaderHTML:     headerHTML,
			FooterHTML:     footerHTML,
			PaperWidth:     paperWidth,
			PaperHeight:    paperHeight,
			MarginTop:      marginTop,
			MarginBottom:   marginBottom,
			MarginLeft:     marginLeft,
			MarginRight:    marginRight,
			Landscape:      landscape,
			RpccBufferSize: googleChromeRpccBufferSize,
		}, nil
	}
	opts, err := resolver()
	if err != nil {
		return opts, xerror.New(op, err)
	}
	return opts, nil
}

func officePrinterOptions(r resource.Resource, config conf.Config) (printer.OfficePrinterOptions, error) {
	const op string = "xhttp.officePrinterOptions"
	resolver := func() (printer.OfficePrinterOptions, error) {
		waitTimeout, err := resource.WaitTimeoutArg(r, config)
		if err != nil {
			return printer.OfficePrinterOptions{}, err
		}
		landscape, err := r.BoolArg(resource.LandscapeArgKey, false)
		if err != nil {
			return printer.OfficePrinterOptions{}, err
		}
		return printer.OfficePrinterOptions{
			WaitTimeout: waitTimeout,
			Landscape:   landscape,
		}, nil
	}
	opts, err := resolver()
	if err != nil {
		return opts, xerror.New(op, err)
	}
	return opts, nil
}
