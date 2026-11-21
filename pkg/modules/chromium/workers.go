package chromium

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// Web workers are targets of their own. No fetch.EventRequestPaused listener
// sees what they load, and the Fetch domain does not exist on them, so with
// --chromium-allow-file-access-from-files (CHROMIUM_ALLOW_FILE_ACCESS_FROM_FILES)
// a worker could read any file:// URL, out of reach of the per-request
// file:// prefixes. Gotenberg starts every worker paused, blocks file:// URLs
// on it, and only then lets it run. A worker never needs a file:// URL: one
// that a file:// page starts without that flag cannot read file:// URLs
// anyway, nor can one that a network page starts. So the lock applies
// whatever the flags.

// dedicatedWorkers selects the targets a page or a worker starts that run
// scripts out of reach of the page's Fetch domain: dedicated workers. Other
// child targets, such as out-of-process iframes, are not auto-attached, as
// they would wait for a debugger that never resumes them.
var dedicatedWorkers = target.Filter{{Type: "worker"}, {Exclude: true}}

// browserWorkers selects the workers no page owns: shared and service
// workers. The browser session auto-attaches them.
var browserWorkers = target.Filter{{Type: "shared_worker"}, {Type: "service_worker"}, {Exclude: true}}

// pauseWorkersOnStart makes the dedicated workers of the target ctx runs
// against start paused, so that [lockWorker] locks each before it runs. It
// replaces chromedp's auto-attach of the target, which lets them run at once.
func pauseWorkersOnStart() chromedp.Action {
	return target.SetAutoAttach(true, true).WithFlatten(true).WithFilter(dedicatedWorkers)
}

// pauseWorkersActionFunc runs [pauseWorkersOnStart] on the page, before it
// navigates.
func pauseWorkersActionFunc(logger *slog.Logger) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		logger.DebugContext(ctx, "start workers paused")

		err := pauseWorkersOnStart().Do(ctx)
		if err != nil {
			return fmt.Errorf("start workers paused: %w", err)
		}

		return nil
	}
}

// lockBrowserWorkers makes Chromium start shared and service workers paused,
// and locks each with [lockWorker] for as long as browserCtx lives. ctx
// bounds the command.
func lockBrowserWorkers(ctx, browserCtx context.Context, logger *slog.Logger) error {
	c := chromedp.FromContext(browserCtx)
	if c == nil || c.Browser == nil {
		return fmt.Errorf("lock workers: %w", chromedp.ErrInvalidContext)
	}

	chromedp.ListenBrowser(browserCtx, func(ev any) {
		if e, ok := ev.(*target.EventAttachedToTarget); ok && isPausedWorker(e) {
			go lockWorker(browserCtx, logger, e.TargetInfo)
		}
	})

	err := target.SetAutoAttach(true, true).WithFlatten(true).WithFilter(browserWorkers).Do(cdp.WithExecutor(ctx, c.Browser))
	if err != nil {
		return fmt.Errorf("start shared and service workers paused: %w", err)
	}

	return nil
}

// listenForWorkers locks with [lockWorker] every worker the target of ctx
// starts paused, see [pauseWorkersOnStart].
func listenForWorkers(ctx context.Context, logger *slog.Logger) {
	chromedp.ListenTarget(ctx, func(ev any) {
		if e, ok := ev.(*target.EventAttachedToTarget); ok && isPausedWorker(e) {
			go lockWorker(ctx, logger, e.TargetInfo)
		}
	})
}

// isPausedWorker reports whether ev attaches a worker that waits for a
// debugger. A target Gotenberg attaches to itself never waits, so locking a
// worker does not lock it again.
func isPausedWorker(ev *target.EventAttachedToTarget) bool {
	if !ev.WaitingForDebugger || ev.TargetInfo == nil {
		return false
	}

	switch ev.TargetInfo.Type {
	case "worker", "shared_worker", "service_worker":
		return true
	default:
		return false
	}
}

// lockWorker blocks file:// URLs on the paused worker info names, makes its
// own dedicated workers start paused, and then lets it run. The worker's
// context derives from parent, and ends with the worker or with parent. If a
// step fails, the worker stays paused: it never runs unlocked.
func lockWorker(parent context.Context, logger *slog.Logger, info *target.Info) {
	workerCtx, cancel := chromedp.NewContext(parent, chromedp.WithTargetID(info.TargetID))

	chromedp.ListenBrowser(workerCtx, func(ev any) {
		if e, ok := ev.(*target.EventTargetDestroyed); ok && e.TargetID == info.TargetID {
			// Not inline: cancel waits for commands this listener would block.
			go cancel()
		}
	})
	listenForWorkers(workerCtx, logger)

	err := chromedp.Run(workerCtx,
		network.SetBlockedURLs([]string{"file://*"}),
		pauseWorkersOnStart(),
		runtime.RunIfWaitingForDebugger(),
	)
	if err != nil {
		if parent.Err() == nil {
			logger.ErrorContext(parent, fmt.Sprintf("lock %s '%s', it stays paused: %s", info.Type, gotenberg.RedactURL(info.URL), err))
		}
		return
	}

	logger.DebugContext(parent, fmt.Sprintf("%s '%s' locked out of file:// URLs", info.Type, gotenberg.RedactURL(info.URL)))
}
