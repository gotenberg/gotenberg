package scenario

import (
	"context"
	"fmt"
	_ "image/png" // Register the PNG decoder for image.DecodeConfig.
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/cucumber/godog"
	"github.com/testcontainers/testcontainers-go"
)

type scenario struct {
	resp               *httptest.ResponseRecorder
	concurrentResps    []*httptest.ResponseRecorder
	probeResps         []*httptest.ResponseRecorder
	sequentialResps    []*httptest.ResponseRecorder
	workdir            string
	teststoreDir       string
	gotenbergContainer testcontainers.Container
	// ownContainer forbids the scenario a pooled default container, and
	// pooledContainer tells that its container came from the pool.
	ownContainer    bool
	pooledContainer bool
	server          *server
	hostPort        int
}

// RunDir is the directory the scenarios of a run work in. TestMain creates and
// removes it.
var RunDir string

// KeepFailedDir, when set, receives the working directory of each failed
// scenario, so that its outputs can be inspected after the run.
var KeepFailedDir string

// cleanupTimeout bounds the cleanup of a scenario.
const cleanupTimeout = time.Minute

// cleanupContext returns a context for cleaning up after a scenario: it keeps
// the values of ctx, but not its cancellation.
func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
}

// keepWorkdir moves the working directory of the failed scenario sc under
// [KeepFailedDir], if set.
func (s *scenario) keepWorkdir(sc *godog.Scenario) {
	if KeepFailedDir == "" || s.workdir == "" {
		return
	}

	name := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, fmt.Sprintf("%s_%s", filepath.Base(sc.Uri), sc.Name))

	err := os.MkdirAll(KeepFailedDir, 0o755)
	if err == nil {
		var dst string
		dst, err = os.MkdirTemp(KeepFailedDir, name+"-")
		if err == nil {
			// os.Rename needs the destination gone; the empty directory only
			// reserved a unique name.
			err = os.Remove(dst)
			if err == nil {
				err = os.Rename(s.workdir, dst)
			}
			if err == nil {
				s.workdir = ""
				fmt.Fprintf(os.Stdout, "kept the working directory of %q in %s\n", sc.Name, dst)
				return
			}
		}
	}

	fmt.Fprintf(os.Stdout, "warning: keep the working directory of %q: %v\n", sc.Name, err)
}

func (s *scenario) reset(ctx context.Context) error {
	s.resp = httptest.NewRecorder()
	s.concurrentResps = nil
	s.probeResps = nil
	s.sequentialResps = nil

	err := os.RemoveAll(s.workdir)
	if err != nil {
		return fmt.Errorf("remove workdir %q: %w", s.workdir, err)
	}
	s.workdir = ""

	if s.server == nil {
		return nil
	}

	err = s.server.stop(ctx)
	if err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	return nil
}

// InitializeScenario registers the steps and hooks of a scenario.
func InitializeScenario(ctx *godog.ScenarioContext) {
	s := &scenario{}
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		// Under the run's directory, which TestMain removes even when a run is
		// interrupted.
		workdir, err := os.MkdirTemp(RunDir, "scenario-")
		if err != nil {
			return ctx, fmt.Errorf("create working directory: %w", err)
		}
		s.workdir = workdir
		s.ownContainer = needsOwnContainer(sc)
		return ctx, nil
	})
	ctx.Given(`^I have a default Gotenberg container$`, s.iHaveADefaultGotenbergContainer)
	ctx.Given(`^I have a Gotenberg container with the following environment variable\(s\):$`, s.iHaveAGotenbergContainerWithTheFollowingEnvironmentVariables)
	ctx.Given(`^I have a (webhook|static) server$`, s.iHaveAServer)
	ctx.When(`^I make a "(GET|HEAD)" request to Gotenberg at the "([^"]*)" endpoint$`, s.iMakeARequestToGotenberg)
	ctx.When(`^I make a "(GET|HEAD)" request to Gotenberg at the "([^"]*)" endpoint with the following header\(s\):$`, s.iMakeARequestToGotenbergWithTheFollowingHeaders)
	ctx.When(`^I make a "(POST)" request to Gotenberg at the "([^"]*)" endpoint with the following form data and header\(s\):$`, s.iMakeARequestToGotenbergWithTheFollowingFormDataAndHeaders)
	ctx.When(`^I make (\d+) concurrent "(POST)" requests to Gotenberg at the "([^"]*)" endpoint with the following form data and header\(s\):$`, s.iMakeConcurrentRequestsToGotenberg)
	ctx.When(`^I make (\d+) sequential "(POST)" requests to Gotenberg at the "([^"]*)" endpoint, probing "([^"]*)" after each, with the following form data and header\(s\):$`, s.iMakeSequentialRequestsToGotenbergProbing)
	ctx.When(`^I wait for the asynchronous request to the webhook$`, s.iWaitForTheAsynchronousRequestToWebhook)
	ctx.When(`^I stop the Gotenberg container$`, s.iStopTheGotenbergContainer)
	ctx.Then(`^the Gotenberg container (should|should NOT) log the following entries:$`, s.theGotenbergContainerShouldLogTheFollowingEntries)
	ctx.Then(`^the response status code should be (\d+)$`, s.theResponseStatusCodeShouldBe)
	ctx.Then(`^all concurrent response status codes should be (\d+)$`, s.allConcurrentResponseStatusCodesShouldBe)
	ctx.Then(`^at least one concurrent response status code should be (\d+)$`, s.someConcurrentResponseStatusCodeShouldBe)
	ctx.Then(`^the (webhook request|file request|server request) method should be "([^"]*)"$`, s.theRequestMethodShouldBe)
	ctx.Then(`^all probe response status codes should be (\d+)$`, s.allProbeResponseStatusCodesShouldBe)
	ctx.Then(`^all sequential response status codes should be (\d+)$`, s.allSequentialResponseStatusCodesShouldBe)
	ctx.Then(`^all concurrent responses should have (\d+) PDF\(s\)$`, s.allConcurrentResponsesShouldHavePdfs)
	ctx.Then(`^the (response|webhook request|file request|server request) header "([^"]*)" should be "([^"]*)"$`, s.theHeaderValueShouldBe)
	ctx.Then(`^the webhook request header "([^"]*)" should carry trace id "([^"]*)"$`, s.theWebhookRequestHeaderShouldCarryTraceID)
	ctx.Then(`^the (response|webhook request|file request|server request) cookie "([^"]*)" should be "([^"]*)"$`, s.theCookieValueShouldBe)
	ctx.Then(`^the (response|webhook request) body should match string:$`, s.theBodyShouldMatchString)
	ctx.Then(`^the (response|webhook request) body should contain string:$`, s.theBodyShouldContainString)
	ctx.Then(`^the (response|webhook request) body should contain the following line\(s\):$`, s.theBodyShouldContainTheFollowingLines)
	ctx.Then(`^the (response|webhook request) body should match JSON:$`, s.theBodyShouldMatchJSON)
	ctx.Then(`^the webhook event should match JSON:$`, s.theWebhookEventShouldMatchJSON)
	ctx.Then(`^there should be (\d+) PDF\(s\) in the (response|webhook request)$`, s.thereShouldBePdfs)
	ctx.Then(`^there should be the following file\(s\) in the (response|webhook request):$`, s.thereShouldBeTheFollowingFiles)
	ctx.Then(`^the (response|webhook request) PDF\(s\) should be valid "([^"]*)" with a tolerance of (\d+) failed rule\(s\)$`, s.thePdfsShouldBeValidWithAToleranceOf)
	ctx.Then(`^the (response|webhook request) PDF\(s\) (should|should NOT) be flatten$`, s.thePdfsShouldBeFlatten)
	ctx.Then(`^the (response|webhook request) PDF\(s\) (should|should NOT) be encrypted`, s.thePdfsShouldBeEncrypted)
	ctx.Then(`^the (response|webhook request) PDF\(s\) (should|should NOT) allow "([^"]*)"$`, s.thePdfsShouldAllowAction)
	ctx.Then(`^the (response|webhook request) PDF\(s\) (should|should NOT) have the "([^"]*)" file embedded$`, s.thePdfsShouldHaveEmbeddedFile)
	ctx.Then(`^the (response|webhook request) PDF\(s\) should have the "([^"]*)" file embedded with relationship "([^"]*)"$`, s.thePdfsShouldHaveEmbeddedFileWithRelationship)
	ctx.Then(`^the (response|webhook request) PDF\(s\) should declare Factur-X XMP with conformance level "([^"]*)"$`, s.thePdfsShouldDeclareFacturXConformanceLevel)
	ctx.Then(`^the "([^"]*)" PDF should have (\d+) page\(s\)$`, s.thePdfShouldHavePages)
	ctx.Then(`^the "([^"]*)" PDF (should|should NOT) be set to landscape orientation$`, s.thePdfShouldBeSetToLandscapeOrientation)
	ctx.Then(`^the "([^"]*)" PDF (should|should NOT) be tagged$`, s.thePdfShouldBeTagged)
	ctx.Then(`^the "([^"]*)" PDF (should|should NOT) have a document outline$`, s.thePdfShouldHaveADocumentOutline)
	ctx.Then(`^the "([^"]*)" PDF (should|should NOT) have the following content at page (\d+):$`, s.thePdfShouldHaveTheFollowingContentAtPage)
	ctx.Then(`^the "([^"]*)" PDF (should|should NOT) have content matching "([^"]*)" at page (\d+)$`, s.thePdfShouldHaveContentMatchingAtPage)
	ctx.Then(`^the "([^"]*)" PDF should have (\d+) image\(s\)$`, s.thePdfShouldHaveImages)
	ctx.Then(`^the "([^"]*)" PDF should reuse (\d+) image XObject\(s\) across (\d+) page\(s\)$`, s.thePdfShouldReuseImageXObjects)
	ctx.Then(`^the (response|webhook request) PDF\(s\) (should|should NOT) be signed$`, s.thePdfsShouldBeSigned)
	ctx.Then(`^the "([^"]*)" PDF should have the "([^"]*)" metadata set to "([^"]*)"$`, s.thePdfShouldHaveMetadata)
	ctx.Then(`^the "([^"]*)" PDF should have the form field "([^"]*)" set to "([^"]*)"$`, s.thePdfShouldHaveFormField)
	ctx.Then(`^the "([^"]*)" PDF page (\d+) should be rotated by (\d+) degrees$`, s.thePdfPageShouldBeRotatedBy)
	ctx.Then(`^the "([^"]*)" PDF first page should be ([\d.]+)x([\d.]+) points$`, s.thePdfShouldHaveSize)
	ctx.Then(`^the "([^"]*)" image should be (\d+)x(\d+) pixels$`, s.theImageShouldBePixels)
	ctx.Then(`^the "([^"]*)" image pixel at (\d+),(\d+) should be "([^"]*)"$`, s.theImagePixelShouldBe)
	ctx.Then(`^the "([^"]*)" file size should be (less|greater) than (\d+) KB$`, s.theFileSizeShouldBe)
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if s.gotenbergContainer == nil {
			return ctx, nil
		}

		// The run's context may be done, on a timeout or an interrupt: the
		// cleanup still gets its own time.
		ctx, cancel := cleanupContext(ctx)
		defer cancel()

		if s.pooledContainer {
			// A failed scenario may leave its container in a bad state.
			if err == nil {
				releaseDefaultContainer(s.gotenbergContainer)
				return ctx, nil
			}

			errDiscard := discardDefaultContainer(ctx, s.gotenbergContainer)
			if errDiscard != nil {
				return ctx, fmt.Errorf("terminate pooled Gotenberg container: %w", errDiscard)
			}
			return ctx, nil
		}

		errTerminate := s.gotenbergContainer.Terminate(ctx, testcontainers.StopTimeout(0))
		if errTerminate != nil {
			return ctx, fmt.Errorf("terminate Gotenberg container: %w", errTerminate)
		}
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if err != nil {
			recordFailedScenario(sc)
			s.keepWorkdir(sc)
		}

		ctx, cancel := cleanupContext(ctx)
		defer cancel()

		errReset := s.reset(ctx)
		if errReset != nil {
			return ctx, fmt.Errorf("reset scenario: %w", errReset)
		}
		return ctx, nil
	})
}
