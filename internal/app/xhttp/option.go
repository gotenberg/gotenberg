package xhttp

import (
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/prinery"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

func chromePrintOptions(r resource.Resource, config conf.Config) (prinery.ChromePrintOptions, error) {
	const op string = "xhttp.chromePrintOptions"
	resolver := func() (prinery.ChromePrintOptions, error) {
		waitDelay, err := resource.WaitDelayArg(r, config)
		if err != nil {
			return prinery.ChromePrintOptions{}, err
		}
		headerHTML, footerHTML,
			err := resource.HeaderFooterContents(r, config)
		if err != nil {
			return prinery.ChromePrintOptions{}, err
		}
		paperWidth, paperHeight,
			err := resource.PaperSizeArgs(r, config)
		if err != nil {
			return prinery.ChromePrintOptions{}, err
		}
		marginTop, marginBottom, marginLeft, marginRight,
			err := resource.MarginArgs(r, config)
		if err != nil {
			return prinery.ChromePrintOptions{}, err
		}
		landscape, err := r.BoolArg(resource.LandscapeArgKey, false)
		if err != nil {
			return prinery.ChromePrintOptions{}, err
		}
		return prinery.ChromePrintOptions{
			WaitDelay:    waitDelay,
			HeaderHTML:   headerHTML,
			FooterHTML:   footerHTML,
			PaperWidth:   paperWidth,
			PaperHeight:  paperHeight,
			MarginTop:    marginTop,
			MarginBottom: marginBottom,
			MarginLeft:   marginLeft,
			MarginRight:  marginRight,
			Landscape:    landscape,
		}, nil
	}
	opts, err := resolver()
	if err != nil {
		return opts, xerror.New(op, err)
	}
	return opts, nil
}

func unoconvPrintOptions(r resource.Resource, config conf.Config) (prinery.UnoconvPrintOptions, error) {
	const op string = "xhttp.unoconvPrintOptions"
	resolver := func() (prinery.UnoconvPrintOptions, error) {
		landscape, err := r.BoolArg(resource.LandscapeArgKey, false)
		if err != nil {
			return prinery.UnoconvPrintOptions{}, err
		}
		return prinery.UnoconvPrintOptions{
			Landscape: landscape,
		}, nil
	}
	opts, err := resolver()
	if err != nil {
		return opts, xerror.New(op, err)
	}
	return opts, nil
}
