package chromium

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/shirou/gopsutil/v4/process"
)

// fakeChromium is the Chromium end of a [devToolsPipe].
type fakeChromium struct {
	commands *bufio.Reader
	messages *os.File
	files    []*os.File
}

func newFakeChromium(t *testing.T) (*devToolsPipe, *fakeChromium) {
	t.Helper()

	pipe, chromiumEnds, err := newDevToolsPipe()
	if err != nil {
		t.Fatalf("newDevToolsPipe: %v", err)
	}
	t.Cleanup(pipe.close)
	t.Cleanup(func() { closeFiles(chromiumEnds...) })

	return pipe, &fakeChromium{
		commands: bufio.NewReader(chromiumEnds[0]),
		messages: chromiumEnds[1],
		files:    chromiumEnds,
	}
}

func (c *fakeChromium) readCommand(t *testing.T) []byte {
	t.Helper()

	command, err := c.commands.ReadBytes(0)
	if err != nil {
		t.Fatalf("read command: %v", err)
	}

	return command[:len(command)-1]
}

func (c *fakeChromium) writeMessage(t *testing.T, message []byte) {
	t.Helper()

	_, err := c.messages.Write(append(message, 0))
	if err != nil {
		t.Fatalf("write message: %v", err)
	}
}

func newTestBridge(t *testing.T, pipe *devToolsPipe) *devToolsBridge {
	t.Helper()

	bridge, err := newDevToolsBridge(pipe, time.Now().Add(10*time.Second))
	if err != nil {
		t.Fatalf("newDevToolsBridge: %v", err)
	}
	t.Cleanup(bridge.closeListener)

	return bridge
}

func dialBridge(t *testing.T, url string, header http.Header) (net.Conn, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := ws.Dialer{}
	if header != nil {
		dialer.Header = ws.HandshakeHeaderHTTP(header)
	}

	conn, _, _, err := dialer.Dial(ctx, url)
	if err == nil {
		t.Cleanup(func() { _ = conn.Close() })
	}

	return conn, err
}

// readServerText reads a text message as chromedp does: without checking
// UTF-8.
func readServerText(conn net.Conn) ([]byte, error) {
	reader := &wsutil.Reader{Source: conn, State: ws.StateClientSide}

	header, err := reader.NextFrame()
	if err != nil {
		return nil, err
	}
	if header.OpCode != ws.OpText {
		return nil, errors.New("not a text frame")
	}

	var message bytes.Buffer
	_, err = message.ReadFrom(reader)

	return message.Bytes(), err
}

func TestDevToolsBridge_relaysMessages(t *testing.T) {
	pipe, chromium := newFakeChromium(t)
	bridge := newTestBridge(t, pipe)

	conn, err := dialBridge(t, bridge.url(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	large := bytes.Repeat([]byte("x"), 8<<20)
	for _, message := range [][]byte{
		[]byte(`{"id":1,"method":"Browser.getVersion"}`),
		// chromedp marshals with invalid UTF-8 allowed.
		[]byte("{\"id\":2,\"params\":{\"value\":\"\xff\"}}"),
		large,
	} {
		err = wsutil.WriteClientText(conn, message)
		if err != nil {
			t.Fatalf("write client text: %v", err)
		}

		got := chromium.readCommand(t)
		if !bytes.Equal(got, message) {
			t.Fatalf("expected Chromium to read %d bytes %.40q, got %d bytes %.40q", len(message), message, len(got), got)
		}

		chromium.writeMessage(t, message)

		got, err = readServerText(conn)
		if err != nil {
			t.Fatalf("read server text: %v", err)
		}
		if !bytes.Equal(got, message) {
			t.Fatalf("expected chromedp to read %d bytes %.40q, got %d bytes %.40q", len(message), message, len(got), got)
		}
	}
}

func TestDevToolsBridge_singleConnection(t *testing.T) {
	pipe, _ := newFakeChromium(t)
	bridge := newTestBridge(t, pipe)

	// The path is the secret: any other one is refused, and the bridge keeps
	// waiting for chromedp.
	wrongPath := bridge.url()[:strings.LastIndex(bridge.url(), "/")+1] + "guess"
	_, err := dialBridge(t, wrongPath, nil)
	if !errors.As(err, new(ws.StatusError)) {
		t.Fatalf("expected a wrong path to be refused, got %v", err)
	}

	// A browser always sends an Origin header; chromedp never does.
	_, err = dialBridge(t, bridge.url(), http.Header{"Origin": []string{"file://"}})
	if !errors.As(err, new(ws.StatusError)) {
		t.Fatalf("expected a request with an Origin header to be refused, got %v", err)
	}

	_, err = dialBridge(t, bridge.url(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Once chromedp is connected, nothing listens anymore.
	address := strings.TrimPrefix(bridge.url(), "ws://")
	address = address[:strings.Index(address, "/")]
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			break
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			t.Fatal("expected the bridge to stop listening once connected")
		}
		time.Sleep(10 * time.Millisecond)
	}

	_, err = dialBridge(t, bridge.url(), nil)
	if err == nil {
		t.Fatal("expected a second connection to fail")
	}
}

func TestDevToolsBridge_closeListenerBeforeConnection(t *testing.T) {
	pipe, _ := newFakeChromium(t)
	bridge := newTestBridge(t, pipe)

	bridge.closeListener()

	_, err := dialBridge(t, bridge.url(), nil)
	if err == nil {
		t.Fatal("expected no connection once the listener is closed")
	}
}

func TestDevToolsBridge_endsWithChromium(t *testing.T) {
	pipe, chromium := newFakeChromium(t)
	bridge := newTestBridge(t, pipe)

	conn, err := dialBridge(t, bridge.url(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Chromium exits: its end of the messages pipe closes.
	closeFiles(chromium.files...)

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = readServerText(conn)
	if err == nil {
		t.Fatal("expected chromedp to lose the connection")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		t.Fatalf("expected the connection to close, got %v", err)
	}
}

func TestDevToolsBridge_endsWithChromedp(t *testing.T) {
	pipe, chromium := newFakeChromium(t)
	bridge := newTestBridge(t, pipe)

	conn, err := dialBridge(t, bridge.url(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	_ = conn.Close()

	// Chromium reads the end of its commands, and exits.
	_, err = chromium.commands.ReadBytes(0)
	if err == nil {
		t.Fatal("expected the commands pipe to close")
	}
}

func TestDevToolsPipe_initialPage(t *testing.T) {
	for _, tc := range []struct {
		scenario  string
		messages  []string
		expectID  string
		expectErr bool
	}{
		{
			scenario: "page reported before the response",
			messages: []string{
				`{"method":"Target.targetCreated","params":{"targetInfo":{"targetId":"UI","type":"browser_ui"}}}`,
				`{"method":"Target.targetCreated","params":{"targetInfo":{"targetId":"PAGE","type":"page"}}}`,
				`{"method":"Target.targetCreated","params":{"targetInfo":{"targetId":"OTHER","type":"page"}}}`,
				`{"id":1,"result":{}}`,
			},
			expectID: "PAGE",
		},
		{
			scenario: "page reported after the response",
			messages: []string{
				`{"id":1,"result":{}}`,
				`{"method":"Target.targetCreated","params":{"targetInfo":{"targetId":"PAGE","type":"page"}}}`,
			},
			expectID: "PAGE",
		},
		{
			scenario:  "error response",
			messages:  []string{`{"id":1,"error":{"code":-32601,"message":"not found"}}`},
			expectErr: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			pipe, chromium := newFakeChromium(t)

			type result struct {
				id  string
				err error
			}
			done := make(chan result, 1)
			go func() {
				id, err := pipe.initialPage()
				done <- result{string(id), err}
			}()

			command := chromium.readCommand(t)
			if !bytes.Contains(command, []byte(`"Target.setDiscoverTargets"`)) {
				t.Fatalf("expected Target.setDiscoverTargets, got %s", command)
			}
			for _, message := range tc.messages {
				chromium.writeMessage(t, []byte(message))
			}

			got := <-done
			if tc.expectErr {
				if got.err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if got.err != nil {
				t.Fatalf("initialPage: %v", got.err)
			}
			if got.id != tc.expectID {
				t.Fatalf("expected page %q, got %q", tc.expectID, got.id)
			}
		})
	}
}

func TestDevToolsPipe_initialPageEndsWithChromium(t *testing.T) {
	pipe, chromium := newFakeChromium(t)

	done := make(chan error, 1)
	go func() {
		_, err := pipe.initialPage()
		done <- err
	}()

	chromium.readCommand(t)
	closeFiles(chromium.files...)

	if err := <-done; err == nil {
		t.Fatal("expected an error once Chromium is gone")
	}
}

// TestChromiumSwitches pins the DevTools transport: a pipe, never a port.
func TestChromiumSwitches(t *testing.T) {
	args := chromiumSwitches(browserArguments{}, "/tmp/profile", "http://127.0.0.1:1234")

	for _, expect := range []string{"--remote-debugging-pipe", "--headless", "--no-sandbox", "--user-data-dir=/tmp/profile", "--proxy-server=http://127.0.0.1:1234", "--disable-features=" + chromiumDisableFeatures} {
		if !strings.Contains(strings.Join(args, " "), expect) {
			t.Errorf("expected %q in %v", expect, args)
		}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "--remote-debugging-port") || strings.HasPrefix(arg, "--disable-component-update") {
			t.Errorf("unexpected %q", arg)
		}
	}
	if args[len(args)-1] != "about:blank" {
		t.Errorf("expected about:blank last, got %q", args[len(args)-1])
	}
}

// startLiveBrowser starts the Chromium or Chrome binary CHROMIUM_BIN_PATH
// names, or skips the test when the variable is unset.
func startLiveBrowser(t *testing.T, arguments browserArguments) *chromiumBrowser {
	t.Helper()

	binPath := os.Getenv("CHROMIUM_BIN_PATH")
	if binPath == "" {
		t.Skip("CHROMIUM_BIN_PATH is not set")
	}

	arguments.binPath = binPath
	arguments.hyphenDataDirPath = t.TempDir()
	arguments.wsUrlReadTimeout = 30 * time.Second

	b := newChromiumBrowser(arguments).(*chromiumBrowser)
	logger := slog.New(slog.DiscardHandler)

	err := b.Start(context.Background(), logger)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		err := b.Stop(logger)
		if err != nil {
			t.Errorf("Stop: %v", err)
		}
	})

	return b
}

// TestChromiumBrowser_live_devToolsPipe starts a real Chromium and checks
// that no DevTools endpoint exists while it converts.
func TestChromiumBrowser_live_devToolsPipe(t *testing.T) {
	b := startLiveBrowser(t, browserArguments{})
	logger := slog.New(slog.DiscardHandler)

	if !b.Healthy(logger) {
		t.Fatal("expected a healthy browser")
	}
	if b.pid == 0 {
		t.Fatal("expected the browser's pid")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	outputPath := filepath.Join(t.TempDir(), "out.png")
	err := b.screenshot(ctx, logger, "data:text/html,<h1>pipe</h1>", outputPath, DefaultScreenshotOptions(), newNetworkAggregate())
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("expected a screenshot, got %v", err)
	}

	_, err = os.Stat(filepath.Join(b.userProfileDirPath, "DevToolsActivePort"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no DevToolsActivePort file, got %v", err)
	}

	p, err := process.NewProcess(int32(b.pid))
	if err != nil {
		t.Fatalf("find Chromium: %v", err)
	}
	processes := []*process.Process{p}
	children, _ := p.Children()
	processes = append(processes, children...)
	for _, p := range processes {
		connections, err := p.Connections()
		if err != nil {
			t.Logf("list connections of %d: %v", p.Pid, err)
			continue
		}
		for _, c := range connections {
			if c.Status == "LISTEN" {
				t.Errorf("expected no listening socket, process %d listens on %s:%d", p.Pid, c.Laddr.IP, c.Laddr.Port)
			}
		}
	}

	pid := b.pid
	err = b.Stop(logger)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("expected Chromium to be gone after Stop, got %v", err)
	}
}

// A Chromium that dies, of the memory watch or of a crash, loses chromedp's
// connection: the browser turns unhealthy, and restarts.
func TestChromiumBrowser_live_restartAfterExit(t *testing.T) {
	b := startLiveBrowser(t, browserArguments{})
	logger := slog.New(slog.DiscardHandler)

	err := syscall.Kill(b.pid, syscall.SIGKILL)
	if err != nil {
		t.Fatalf("kill Chromium: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for b.ctx.Err() == nil {
		if time.Now().After(deadline) {
			t.Fatal("expected the browser context to end once Chromium is gone")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if b.Healthy(logger) {
		t.Fatal("expected an unhealthy browser once Chromium is gone")
	}

	err = b.Stop(logger)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	err = b.Start(context.Background(), logger)
	if err != nil {
		t.Fatalf("Start after exit: %v", err)
	}
	if !b.Healthy(logger) {
		t.Fatal("expected a healthy browser after the restart")
	}
}
