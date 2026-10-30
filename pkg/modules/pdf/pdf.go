package pdf

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dlclark/regexp2/v2"
	litho "github.com/gotenberg/litho/pkg/pdf"
	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(PDF))
}

// PDF processes PDF documents with litho. It runs [Stages] for the Chromium
// and LibreOffice routes through [Provider], and exposes the /forms/pdf
// routes.
type PDF struct {
	processor      *Processor
	maxConcurrency int
	disableRoutes  bool
	allowList      []*regexp2.Regexp
	denyList       []*regexp2.Regexp
	denyPrivateIPs bool
	denyPublicIPs  bool
	envProxy       bool
}

// Provider gives access to the [Processor] of the PDF module.
//
//	provider, _ := ctx.Module(new(pdf.Provider))
//	processor := provider.(pdf.Provider).Processor()
type Provider interface {
	Processor() *Processor
}

// Descriptor returns the PDF module's descriptor.
func (mod *PDF) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "pdf",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("pdf", flag.ExitOnError)
			fs.Int("pdf-max-concurrency", defaultMaxConcurrency, "Set the maximum number of PDF documents a request processes concurrently, across all requests - raising it trades memory and CPU for speed on multi-document requests")
			fs.String("pdf-max-stream-bytes", "", "Set the maximum size a PDF stream may decode to - it accepts values like 256MB, 1GB, etc; empty keeps litho's default")
			fs.String("pdf-max-embedded-file-bytes", "", "Set the maximum size of an embedded file read for validation - it accepts values like 50MB, 1GB, etc; empty keeps litho's default")
			fs.Int64("pdf-max-image-pixels", 0, "Set the maximum number of pixels an image may hold - 0 keeps litho's default")
			fs.StringSlice("pdf-allow-list", []string{}, `Set the allowed timestamp authority URLs for the sign and timestamp stages using regular expressions - supports multiple values. A match bypasses --pdf-deny-private-ips (PDF_DENY_PRIVATE_IPS) and --pdf-deny-public-ips (PDF_DENY_PUBLIC_IPS), so terminate the host or the pattern also matches suffix hosts, for example ^https?://tsa\.example\.com(:|/|$)`)
			fs.StringSlice("pdf-deny-list", []string{}, "Set the denied timestamp authority URLs for the sign and timestamp stages using regular expressions - supports multiple values")
			fs.Bool("pdf-deny-private-ips", false, "Reject timestamp authority URLs whose host resolves to a non-public IP address (loopback, RFC1918, link-local, unique-local). Enable on deployments that accept untrusted requests to mitigate SSRF against internal services")
			fs.Bool("pdf-deny-public-ips", false, "Reject timestamp authority URLs whose host resolves to a public IP address. Enable on air-gapped or data-governed deployments")
			fs.Bool("pdf-enable-environment-proxy", false, "Route timestamp authority requests through the proxy defined by the standard HTTP_PROXY, HTTPS_PROXY, and NO_PROXY variables, including credentials")
			fs.Bool("pdf-disable-routes", false, "Disable the routes")

			return fs
		}(),
		New: func() gotenberg.Module { return new(PDF) },
	}
}

// Provision sets the module properties.
func (mod *PDF) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.maxConcurrency = flags.MustInt("pdf-max-concurrency")
	mod.disableRoutes = flags.MustBool("pdf-disable-routes")
	mod.allowList = flags.MustRegexpSlice("pdf-allow-list")
	mod.denyList = flags.MustRegexpSlice("pdf-deny-list")
	mod.denyPrivateIPs = flags.MustBool("pdf-deny-private-ips")
	mod.denyPublicIPs = flags.MustBool("pdf-deny-public-ips")
	mod.envProxy = flags.MustBool("pdf-enable-environment-proxy")

	mod.processor = &Processor{
		limits: litho.Limits{
			MaxStreamBytes:       flags.MustHumanReadableBytes("pdf-max-stream-bytes"),
			MaxEmbeddedFileBytes: flags.MustHumanReadableBytes("pdf-max-embedded-file-bytes"),
			MaxImagePixels:       flags.MustInt64("pdf-max-image-pixels"),
		},
		// Remediation embeds the fonts a document shows without embedding them.
		// Built once, so the font directories are indexed once per process.
		fonts:      litho.WithSystemFonts(),
		httpClient: mod.httpClient,
		files:      newFileConcurrency(mod.maxConcurrency),
	}

	return nil
}

// Validate validates the module properties.
func (mod *PDF) Validate() error {
	if mod.maxConcurrency < 1 {
		return fmt.Errorf("PDF max concurrency must be at least 1, got %d; set --pdf-max-concurrency (env PDF_MAX_CONCURRENCY) to a positive value", mod.maxConcurrency)
	}

	err := mod.processor.limits.Validate()
	if err != nil {
		return fmt.Errorf("invalid PDF limits; check --pdf-max-stream-bytes (env PDF_MAX_STREAM_BYTES), --pdf-max-embedded-file-bytes (env PDF_MAX_EMBEDDED_FILE_BYTES) and --pdf-max-image-pixels (env PDF_MAX_IMAGE_PIXELS): %w", err)
	}

	return nil
}

// SystemMessages returns the module's settings.
func (mod *PDF) SystemMessages() []string {
	return []string{
		fmt.Sprintf("max concurrency - %d", mod.maxConcurrency),
	}
}

// Processor returns the [Processor] that runs [Stages].
func (mod *PDF) Processor() *Processor {
	return mod.processor
}

// httpClient returns the client litho reaches timestamp authorities with,
// bounded by the request's deadline and filtered by the module's allow and
// deny lists.
func (mod *PDF) httpClient(ctx context.Context) *http.Client {
	timeout := time.Minute
	deadline, ok := ctx.Deadline()
	if ok {
		timeout = time.Until(deadline)
	}

	return gotenberg.NewOutboundHttpClient(
		timeout,
		mod.allowList,
		mod.denyList,
		mod.envProxy,
		gotenberg.WithDenyPrivateIPs(mod.denyPrivateIPs),
		gotenberg.WithDenyPublicIPs(mod.denyPublicIPs),
	)
}

// Routes returns the HTTP routes.
func (mod *PDF) Routes() ([]api.Route, error) {
	if mod.disableRoutes {
		return nil, nil
	}

	routes := []api.Route{
		pipeRoute(mod.processor),
		metadataRoute(mod.processor),
		bookmarksRoute(mod.processor),
		infoRoute(mod.processor),
		validateRoute(mod.processor),
		attachmentsRoute(mod.processor),
	}

	for _, op := range sortedKeys(stageFactories) {
		routes = append(routes, stageRoute(mod.processor, op))
	}

	return routes, nil
}

// Interface guards.
var (
	_ gotenberg.Module       = (*PDF)(nil)
	_ gotenberg.Provisioner  = (*PDF)(nil)
	_ gotenberg.Validator    = (*PDF)(nil)
	_ gotenberg.SystemLogger = (*PDF)(nil)
	_ Provider               = (*PDF)(nil)
	_ api.Router             = (*PDF)(nil)
)
