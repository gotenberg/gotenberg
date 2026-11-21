package chromium

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// devToolsPipe is Gotenberg's end of Chromium's --remote-debugging-pipe.
// Chromium reads DevTools commands from its file descriptor 3 and writes
// responses and events to its file descriptor 4, each message a JSON text
// ended by a NUL byte. No port carries the protocol, so no page Chromium
// renders can reach it.
type devToolsPipe struct {
	// commands is the write end of Chromium's file descriptor 3.
	commands *os.File
	// messages is the read end of Chromium's file descriptor 4.
	messages *os.File
	reader   *bufio.Reader

	closeOnce sync.Once
}

// newDevToolsPipe returns the pipe, and the files Chromium inherits as its
// file descriptors 3 and 4, in that order, for [exec.Cmd.ExtraFiles]. The
// caller closes these files once Chromium has started.
func newDevToolsPipe() (*devToolsPipe, []*os.File, error) {
	chromiumCommands, commands, err := os.Pipe()
	if err != nil {
		return nil, nil, fmt.Errorf("create commands pipe: %w", err)
	}

	messages, chromiumMessages, err := os.Pipe()
	if err != nil {
		_ = chromiumCommands.Close()
		_ = commands.Close()
		return nil, nil, fmt.Errorf("create messages pipe: %w", err)
	}

	p := &devToolsPipe{
		commands: commands,
		messages: messages,
		reader:   bufio.NewReader(messages),
	}

	return p, []*os.File{chromiumCommands, chromiumMessages}, nil
}

// send writes one message to Chromium. It is not safe for concurrent use.
func (p *devToolsPipe) send(message []byte) error {
	_, err := p.commands.Write(append(message, 0))
	if err != nil {
		return fmt.Errorf("write to Chromium: %w", err)
	}

	return nil
}

// receive reads the next message from Chromium, without its NUL byte. It is
// not safe for concurrent use.
func (p *devToolsPipe) receive() ([]byte, error) {
	message, err := p.reader.ReadBytes(0)
	if err != nil {
		return nil, fmt.Errorf("read from Chromium: %w", err)
	}

	return message[:len(message)-1], nil
}

// close closes both ends Gotenberg holds. Chromium exits once it reads the
// end of its commands, and a pending receive returns.
func (p *devToolsPipe) close() {
	p.closeOnce.Do(func() {
		_ = p.commands.Close()
		_ = p.messages.Close()
	})
}

// initialPage turns target discovery on and returns the ID of the first page
// target Chromium reports: the about:blank tab it opens at start. The browser
// context attaches to that tab, rather than opening a second one, and target
// discovery stays on for the browser session, as chromedp's own exec
// allocator leaves it. It must run before the bridge relays the pipe.
func (p *devToolsPipe) initialPage() (target.ID, error) {
	const id = 1

	err := p.send([]byte(fmt.Sprintf(`{"id":%d,"method":"Target.setDiscoverTargets","params":{"discover":true}}`, id)))
	if err != nil {
		return "", err
	}

	var (
		acknowledged bool
		page         target.ID
	)

	for !acknowledged || page == "" {
		raw, err := p.receive()
		if err != nil {
			return "", err
		}

		var message struct {
			ID     int64  `json:"id"`
			Method string `json:"method"`
			Params struct {
				TargetInfo struct {
					TargetID target.ID `json:"targetId"`
					Type     string    `json:"type"`
				} `json:"targetInfo"`
			} `json:"params"`
			Error *struct {
				Code    int64  `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		err = json.Unmarshal(raw, &message)
		if err != nil {
			return "", fmt.Errorf("decode message from Chromium: %w", err)
		}

		switch {
		case message.ID == id && message.Error != nil:
			return "", fmt.Errorf("discover targets: %s (%d)", message.Error.Message, message.Error.Code)
		case message.ID == id:
			acknowledged = true
		case message.Method == "Target.targetCreated" && message.Params.TargetInfo.Type == "page" && page == "":
			page = message.Params.TargetInfo.TargetID
		}
	}

	return page, nil
}

// devToolsBridge serves a [devToolsPipe] to chromedp, which only speaks
// WebSocket, as a single WebSocket connection. It listens on a loopback port
// under a path of 256 random bits, accepts the first client that asks for
// that path, and closes its listener right away: once chromedp is connected,
// nothing listens anymore, and no other client ever reaches Chromium.
type devToolsBridge struct {
	pipe     *devToolsPipe
	listener net.Listener
	path     string
	deadline time.Time

	closeListener func()
}

// newDevToolsBridge starts listening for chromedp. The upgrade of each
// connection must complete before deadline. The caller calls closeListener
// once chromedp is connected or gave up.
func newDevToolsBridge(pipe *devToolsPipe, deadline time.Time) (*devToolsBridge, error) {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, fmt.Errorf("generate DevTools path: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for chromedp: %w", err)
	}

	b := &devToolsBridge{
		pipe:     pipe,
		listener: listener,
		path:     "/devtools/browser/" + hex.EncodeToString(secret),
		deadline: deadline,
	}
	b.closeListener = sync.OnceFunc(func() { _ = listener.Close() })

	go b.serve()

	return b, nil
}

// url returns the WebSocket URL chromedp dials.
func (b *devToolsBridge) url() string {
	return "ws://" + b.listener.Addr().String() + b.path
}

// serve accepts connections until one upgrades on the bridge's path, then
// closes the listener and relays that connection until either side ends.
func (b *devToolsBridge) serve() {
	upgrader := ws.Upgrader{
		OnRequest: func(uri []byte) error {
			if string(uri) != b.path {
				return ws.RejectConnectionError(ws.RejectionStatus(http.StatusNotFound))
			}
			return nil
		},
		OnHeader: func(key, _ []byte) error {
			// chromedp sends no Origin header; a browser always does.
			if bytes.EqualFold(key, []byte("Origin")) {
				return ws.RejectConnectionError(ws.RejectionStatus(http.StatusForbidden))
			}
			return nil
		},
	}

	for {
		conn, err := b.listener.Accept()
		if err != nil {
			return
		}

		_ = conn.SetDeadline(b.deadline)
		_, err = upgrader.Upgrade(conn)
		if err != nil {
			_ = conn.Close()
			continue
		}
		_ = conn.SetDeadline(time.Time{})

		b.closeListener()
		b.relay(conn)

		return
	}
}

// relay copies the WebSocket text frames of conn to the pipe, and the pipe's
// messages to conn as text frames, until either side ends. It then closes
// both, so that chromedp sees a lost connection once Chromium is gone, and
// Chromium exits once chromedp is gone.
func (b *devToolsBridge) relay(conn net.Conn) {
	defer b.pipe.close()
	defer func() { _ = conn.Close() }()

	go func() {
		// Closing the connection also ends the read loop below.
		defer func() { _ = conn.Close() }()

		w := bufio.NewWriter(conn)
		for {
			message, err := b.pipe.receive()
			if err != nil {
				return
			}

			// One frame per message: chromedp does not reassemble fragments.
			err = wsutil.WriteServerText(w, message)
			if err == nil {
				err = w.Flush()
			}
			if err != nil {
				return
			}
		}
	}()

	// chromedp marshals with invalid UTF-8 allowed, so the reader does not
	// check it. Control frames get no answer: chromedp sends no ping, and a
	// close frame ends the relay.
	reader := &wsutil.Reader{
		Source: bufio.NewReader(conn),
		State:  ws.StateServerSide,
	}

	var message bytes.Buffer
	for {
		header, err := reader.NextFrame()
		if err != nil {
			return
		}

		switch {
		case header.OpCode == ws.OpClose:
			return
		case header.OpCode != ws.OpText:
			err = reader.Discard()
			if err != nil {
				return
			}
			continue
		}

		message.Reset()
		_, err = message.ReadFrom(reader)
		if err != nil && !errors.Is(err, io.EOF) {
			return
		}

		err = b.pipe.send(message.Bytes())
		if err != nil {
			return
		}
	}
}
