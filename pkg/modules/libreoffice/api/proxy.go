package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// libreOfficeProxy is the [gotenberg.OutboundProxy] LibreOffice routes its
// outbound requests through, so the same allow/deny lists and IP-class filters
// that protect Chromium and webhook fetches also apply to LibreOffice's own
// libcurl-driven fetches.
//
// The profile ([profileConfigTmpl]) keeps LibreOffice from loading the
// images and other content an uploaded document links to. What still reaches
// the proxy: the style sheets an HTML document links (<link
// rel="stylesheet">), and what a document's macros fetch when
// --libreoffice-enable-macros (LIBREOFFICE_ENABLE_MACROS) runs them. Without a
// filtering proxy in the path those requests would bypass every Go-side SSRF
// guard, because they originate inside the lok subprocess.
type libreOfficeProxy struct {
	outbound *gotenberg.OutboundProxy
	logger   *slog.Logger

	// conversionMu guards conversion, the client request lok converts a
	// document for: the proxy logs its refusals with it. See [attach].
	conversionMu sync.Mutex
	conversion   *proxyConversion
}

// proxyConversion is the client request a conversion runs for.
type proxyConversion struct {
	ctx    context.Context
	logger *slog.Logger
}

// newLibreOfficeProxy returns a proxy that applies opts to every request
// LibreOffice sends. Callers must call [libreOfficeProxy.Start] before
// pointing LibreOffice at the proxy and [libreOfficeProxy.Stop] on shutdown.
func newLibreOfficeProxy(logger *slog.Logger, opts gotenberg.OutboundProxyOptions) *libreOfficeProxy {
	p := &libreOfficeProxy{
		logger: logger.With(slog.String("logger", "libreoffice-proxy")),
	}
	opts.LogRefusal = p.logRefusal
	p.outbound = gotenberg.NewOutboundProxy(opts)

	return p
}

// Start binds the proxy to a free local port and serves in the background
// until [libreOfficeProxy.Stop] is called.
func (p *libreOfficeProxy) Start() error {
	return p.outbound.Start(p.logger)
}

// Stop shuts the proxy down, closing its tunnels. Subsequent calls are no-ops.
func (p *libreOfficeProxy) Stop(ctx context.Context) error {
	return p.outbound.Stop(ctx)
}

// Addr returns the host:port the proxy listens on.
func (p *libreOfficeProxy) Addr() string {
	return p.outbound.Addr()
}

// attach makes the proxy log its refusals with ctx and logger, the client
// request's, so that the records carry its correlation ID and trace, until
// the returned function runs. lok converts one document at a time, so the
// requests the proxy sees meanwhile are that conversion's.
func (p *libreOfficeProxy) attach(ctx context.Context, logger *slog.Logger) func() {
	p.conversionMu.Lock()
	defer p.conversionMu.Unlock()

	p.conversion = &proxyConversion{ctx: ctx, logger: logger}

	return func() {
		p.conversionMu.Lock()
		defer p.conversionMu.Unlock()

		p.conversion = nil
	}
}

// logRefusal records a request to rawURL the filters refused, with the
// attached conversion's context and logger, or fallback and the proxy's own
// logger outside a conversion.
func (p *libreOfficeProxy) logRefusal(fallback context.Context, rawURL string, err error) {
	ctx, logger := fallback, p.logger

	p.conversionMu.Lock()
	if p.conversion != nil {
		ctx, logger = p.conversion.ctx, p.conversion.logger
	}
	p.conversionMu.Unlock()

	gotenberg.LogOutboundRefusal(ctx, logger, "outbound request blocked by the LibreOffice filters", rawURL, err)
}

// profileConfigTmpl is the registrymodifications.xcu LibreOffice loads at
// startup, in the lok process. It does three things:
//
//   - Routes every HTTP and HTTPS fetch through proxyHost:proxyPort so
//     LibreOffice's own libcurl fetches hit the in-process SSRF proxy.
//   - Sets BlockUntrustedRefererLinks so LibreOffice refuses to load content
//     linked from a document that sits in an untrusted location.
//   - Never updates a document's links on load, in Writer (Link 2) and Calc
//     (Link 1, as Calc numbers the choices differently). LibreOfficeKit cannot
//     set the load-time update mode itself, so the profile does.
//
// The second and third settings close the local-read and direct-fetch vectors
// the proxy cannot see. A document that links an absolute path
// (file:///etc/...) or any URL is loaded from the per-request temp dir,
// which is never a trusted location, so LibreOffice drops the linked content
// instead of resolving it. Embedded content (stored inside the document)
// is unaffected. The HTML import still requests the style sheets a page links,
// and a document's macros, once --libreoffice-enable-macros
// (LIBREOFFICE_ENABLE_MACROS) runs them, may open URLs: those requests go
// through the proxy.
//
// The %s placeholders accept the proxy host and port respectively (host
// first, port second, repeated for HTTP and HTTPS).
const profileConfigTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<oor:items xmlns:oor="http://openoffice.org/2001/registry" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <item oor:path="/org.openoffice.Office.Common/Security/Scripting"><prop oor:name="BlockUntrustedRefererLinks" oor:op="fuse"><value>true</value></prop></item>
  <item oor:path="/org.openoffice.Office.Writer/Content/Update"><prop oor:name="Link" oor:op="fuse"><value>2</value></prop></item>
  <item oor:path="/org.openoffice.Office.Calc/Content/Update"><prop oor:name="Link" oor:op="fuse"><value>1</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetProxyType" oor:op="fuse"><value>1</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetHTTPProxyName" oor:op="fuse"><value>%s</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetHTTPProxyPort" oor:op="fuse"><value>%s</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetHTTPSProxyName" oor:op="fuse"><value>%s</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetHTTPSProxyPort" oor:op="fuse"><value>%s</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetNoProxy" oor:op="fuse"><value></value></prop></item>
</oor:items>
`

// writeProfileConfig drops a registrymodifications.xcu file into
// userProfileDirPath/user/ that points LibreOffice's UCB layer at proxyAddr
// for both HTTP and HTTPS, blocks linked content from untrusted locations and
// never updates links on load. proxyAddr must be a host:port pair.
func writeProfileConfig(userProfileDirPath, proxyAddr string) error {
	host, port, err := net.SplitHostPort(proxyAddr)
	if err != nil {
		return fmt.Errorf("split proxy address %q: %w", proxyAddr, err)
	}

	userDir := userProfileDirPath + "/user"
	err = os.MkdirAll(userDir, 0o755)
	if err != nil {
		return fmt.Errorf("create LibreOffice user profile directory: %w", err)
	}

	body := fmt.Sprintf(profileConfigTmpl, host, port, host, port)
	err = os.WriteFile(userDir+"/registrymodifications.xcu", []byte(body), 0o600)
	if err != nil {
		return fmt.Errorf("write registrymodifications.xcu: %w", err)
	}

	return nil
}

// proxyEnv overlays http_proxy/https_proxy on env so LibreOffice's libcurl
// path also routes through proxyAddr. LibreOffice runs inside the lok process,
// which inherits them. The environment variables supplement the
// registrymodifications.xcu config so coverage stays intact if LibreOffice
// upgrades and one of the two paths regresses.
func proxyEnv(env []string, proxyAddr string) []string {
	proxyURL := "http://" + proxyAddr

	filtered := env[:0:0]
	for _, kv := range env {
		switch strings.ToLower(strings.SplitN(kv, "=", 2)[0]) {
		case "http_proxy", "https_proxy", "no_proxy":
			continue
		}
		filtered = append(filtered, kv)
	}

	return append(filtered,
		"http_proxy="+proxyURL,
		"https_proxy="+proxyURL,
		"HTTP_PROXY="+proxyURL,
		"HTTPS_PROXY="+proxyURL,
		"no_proxy=",
		"NO_PROXY=",
	)
}
