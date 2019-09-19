package xhttp

import (
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/print"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

func chromePrintOptions(r resource.Resource, config conf.Config) (print.ChromePrintOptions, error) {
	const op string = "xhttp.chromePrintOptions"
	resolver := func() (print.ChromePrintOptions, error) {
		waitDelay, err := resource.WaitDelayArg(r, config)
		if err != nil {
			return print.ChromePrintOptions{}, err
		}
		headerHTML, footerHTML,
			err := resource.HeaderFooterContents(r, config)
		if err != nil {
			return print.ChromePrintOptions{}, err
		}
		paperWidth, paperHeight,
			err := resource.PaperSizeArgs(r, config)
		if err != nil {
			return print.ChromePrintOptions{}, err
		}
		marginTop, marginBottom, marginLeft, marginRight,
			err := resource.MarginArgs(r, config)
		if err != nil {
			return print.ChromePrintOptions{}, err
		}
		landscape, err := r.BoolArg(resource.LandscapeArgKey, false)
		if err != nil {
			return print.ChromePrintOptions{}, err
		}
		return print.ChromePrintOptions{
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

func officePrintOptions(r resource.Resource, config conf.Config) (print.OfficePrintOptions, error) {
	const op string = "xhttp.officePrintOptions"
	resolver := func() (print.OfficePrintOptions, error) {
		landscape, err := r.BoolArg(resource.LandscapeArgKey, false)
		if err != nil {
			return print.OfficePrintOptions{}, err
		}
		return print.OfficePrintOptions{
			Landscape: landscape,
		}, nil
	}
	opts, err := resolver()
	if err != nil {
		return opts, xerror.New(op, err)
	}
	return opts, nil
}
