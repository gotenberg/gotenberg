package chromium

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

func TestIsPausedWorker(t *testing.T) {
	for _, tc := range []struct {
		targetType string
		waiting    bool
		expect     bool
	}{
		{targetType: "worker", waiting: true, expect: true},
		{targetType: "shared_worker", waiting: true, expect: true},
		{targetType: "service_worker", waiting: true, expect: true},
		// chromedp's own attach to a worker never waits: locking it again
		// would loop.
		{targetType: "worker", waiting: false, expect: false},
		{targetType: "iframe", waiting: true, expect: false},
		{targetType: "page", waiting: true, expect: false},
	} {
		t.Run(tc.targetType, func(t *testing.T) {
			ev := &target.EventAttachedToTarget{
				TargetInfo:         &target.Info{Type: tc.targetType},
				WaitingForDebugger: tc.waiting,
			}
			if got := isPausedWorker(ev); got != tc.expect {
				t.Fatalf("isPausedWorker(%s, waiting %t) = %t, want %t", tc.targetType, tc.waiting, got, tc.expect)
			}
		})
	}

	if isPausedWorker(&target.EventAttachedToTarget{WaitingForDebugger: true}) {
		t.Fatal("expected an event without target info not to be a paused worker")
	}
}

// workersPage reads a file:// marker outside the page's directory from a
// dedicated, a nested, a shared and a file:// worker, reads a file of its own
// directory, and embeds a cross-site iframe. It writes what each read in its
// title.
const workersPage = `<html><body><script>
var target = location.href.replace('req/index.html', 'other/marker.txt');
var rd = "function rd(){var x=new XMLHttpRequest();try{x.open('GET','" + target + "',false);x.send();return 'READ:'+x.responseText}catch(e){return 'BLOCKED'}}";
var results = [];
var want = 6;
function done(s){results.push(s);if(results.length==want)document.title=results.join(' | ');}
onmessage = function(e){done(e.data)};
var dedicated = new Worker(URL.createObjectURL(new Blob([rd + ";postMessage('DEDICATED:'+rd())"])));
dedicated.onmessage = function(e){done(e.data)};
var inner = rd + ";postMessage('NESTED:'+rd())";
var outer = new Worker(URL.createObjectURL(new Blob([rd + ";var w=new Worker(URL.createObjectURL(new Blob([" + JSON.stringify(inner) + "])));w.onmessage=function(e){postMessage(e.data)}"])));
outer.onmessage = function(e){done(e.data)};
var shared = new SharedWorker(URL.createObjectURL(new Blob([rd + ";onconnect=function(e){e.ports[0].postMessage('SHARED:'+rd())}"])));
shared.port.onmessage = function(e){done(e.data)};
var file = new Worker('worker.js');
file.onmessage = function(e){done(e.data)};
var x = new XMLHttpRequest();
try{x.open('GET','own.txt',false);x.send();done('OWN:'+x.responseText.trim())}catch(e){done('OWN:BLOCKED')}
setTimeout(function(){for(var i=0;i<want;i++)done('TIMEOUT')},10000);
</script><iframe src="FRAME_URL"></iframe></body></html>`

// workersScript is a worker Chromium loads from a file:// URL.
const workersScript = `var x=new XMLHttpRequest();
try{x.open('GET',location.href.replace('req/worker.js','other/marker.txt'),false);x.send();postMessage('FILE:READ:'+x.responseText)}catch(e){postMessage('FILE:BLOCKED')}`

// TestChromiumBrowser_live_workersLockedOutOfFiles converts, with
// --chromium-allow-file-access-from-files, a page whose workers try to read a
// file outside the page's directory.
func TestChromiumBrowser_live_workersLockedOutOfFiles(t *testing.T) {
	for _, sharedBrowserContext := range []bool{false, true} {
		t.Run(fmt.Sprintf("shared browser context %t", sharedBrowserContext), func(t *testing.T) {
			b := startLiveBrowser(t, browserArguments{allowFileAccessFromFiles: true, sharedBrowserContext: sharedBrowserContext})
			logger := slog.New(slog.DiscardHandler)

			frame := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`<script>parent.postMessage('FRAME','*')</script>`))
			}))
			defer frame.Close()

			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatalf("resolve temporary directory: %v", err)
			}
			for name, content := range map[string]string{
				"req/index.html":   strings.Replace(workersPage, "FRAME_URL", frame.URL, 1),
				"req/worker.js":    workersScript,
				"req/own.txt":      "OWN-OK",
				"other/marker.txt": "MARKER",
			} {
				path := filepath.Join(root, name)
				err = os.MkdirAll(filepath.Dir(path), 0o755)
				if err == nil {
					err = os.WriteFile(path, []byte(content), 0o600)
				}
				if err != nil {
					t.Fatalf("write %s: %v", name, err)
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			options := DefaultOptions()
			options.AllowedFilePrefixes = []string{filepath.Join(root, "req")}

			var title string
			err = b.do(ctx, logger, "file://"+filepath.Join(root, "req", "index.html"), options, newNetworkAggregate(), chromedp.Tasks{
				network.Enable(),
				fetch.Enable(),
				runtime.Enable(),
				pauseWorkersActionFunc(logger),
				chromedp.Navigate("file://" + filepath.Join(root, "req", "index.html")),
				chromedp.Poll("document.title.length > 0", nil),
				chromedp.Title(&title),
			})
			if err != nil {
				t.Fatalf("do: %v", err)
			}

			if strings.Contains(title, "MARKER") || strings.Contains(title, "TIMEOUT") {
				t.Fatalf("expected no worker to read the marker, got %q", title)
			}
			for _, expect := range []string{"DEDICATED:BLOCKED", "NESTED:BLOCKED", "SHARED:BLOCKED", "FILE:BLOCKED", "OWN:OWN-OK", "FRAME"} {
				if !strings.Contains(title, expect) {
					t.Errorf("expected %q in %q", expect, title)
				}
			}
		})
	}
}
