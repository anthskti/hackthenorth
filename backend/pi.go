package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// PiClient talks to QNX kvmd over HTTP (see embedded/qnx/kvmd.c).
// POST /key body is Leonardo lines, one per newline: t<text> p<hex> r<hex> k<hex> a
type PiClient struct {
	base        string
	stream      string
	key         string
	snapshot    string
	streamWidth  int
	streamHeight int
	http         *http.Client
	sendMu       sync.Mutex
}

func NewPiClient() *PiClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("QNX_BASE_URL")), "/")
	stream := strings.TrimSpace(os.Getenv("QNX_STREAM_URL"))
	key := strings.TrimSpace(os.Getenv("QNX_KEY_URL"))
	snap := strings.TrimSpace(os.Getenv("QNX_SNAPSHOT_URL"))
	if stream == "" && base != "" {
		stream = base + "/stream"
	}
	if key == "" && base != "" {
		key = base + "/key"
	}
	if snap == "" && base != "" {
		snap = base + "/snapshot.jpg"
	}
	sw, sh := streamDimensionsFromEnv()
	return &PiClient{
		base:         base,
		stream:       stream,
		key:          key,
		snapshot:     snap,
		streamWidth:  sw,
		streamHeight: sh,
		http:         &http.Client{Timeout: 4 * time.Second},
	}
}

func streamDimensionsFromEnv() (width, height int) {
	width = 854
	height = 480
	if v := strings.TrimSpace(os.Getenv("QNX_STREAM_WIDTH")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			width = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("QNX_STREAM_HEIGHT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			height = n
		}
	}
	return width, height
}

func (p *PiClient) Enabled() bool {
	return p != nil && p.stream != ""
}

func (p *PiClient) KeyURL() string {
	if p == nil {
		return ""
	}
	return p.key
}

func (p *PiClient) Probe(ctx context.Context) error {
	if p.snapshot == "" {
		return fmt.Errorf("QNX snapshot URL is not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.snapshot, nil)
	if err != nil {
		return err
	}
	res, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64))
	if res.StatusCode >= 400 {
		return fmt.Errorf("qnx snapshot %s", res.Status)
	}
	return nil
}

func (p *PiClient) Snapshot(ctx context.Context) ([]byte, error) {
	if p.snapshot == "" {
		return nil, fmt.Errorf("QNX snapshot URL is not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.snapshot, nil)
	if err != nil {
		return nil, err
	}
	res, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("qnx snapshot %s: %s", res.Status, truncate(string(body), 200))
	}
	if len(body) < 2 || body[0] != 0xff || body[1] != 0xd8 {
		return nil, fmt.Errorf("qnx snapshot was not a JPEG")
	}
	return body, nil
}

// SendRawKeyBody forwards Leonardo lines to QNX POST /key unchanged (same as Pi /ui).
func (p *PiClient) SendRawKeyBody(ctx context.Context, body string) error {
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	if p.key == "" {
		return fmt.Errorf("QNX key URL is not set")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("empty key body")
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.key, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	res, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	if res.StatusCode >= 400 {
		return fmt.Errorf("qnx /key %s: %s", res.Status, truncate(string(respBody), 200))
	}
	return nil
}

func (p *PiClient) SendCommands(ctx context.Context, cmds ...string) error {
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	if p.key == "" {
		return fmt.Errorf("QNX key URL is not set")
	}
	var lines []string
	for _, c := range cmds {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		lines = append(lines, c)
	}
	if len(lines) == 0 {
		return fmt.Errorf("no keyboard commands")
	}
	body := strings.Join(lines, "\n") + "\n"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.key, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	res, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	if res.StatusCode >= 400 {
		return fmt.Errorf("qnx /key %s: %s", res.Status, truncate(string(respBody), 200))
	}
	return nil
}

func (p *PiClient) SendKey(ctx context.Context, value string) error {
	cmd, err := valueToKVMCommand(value)
	if err != nil {
		return err
	}
	return p.SendCommands(ctx, cmd)
}

func decisionToKVMCommands(d *AgentDecision) ([]string, error) {
	if d == nil {
		return nil, fmt.Errorf("no decision")
	}
	switch d.Action {
	case "done":
		return nil, nil
	case "wait":
		return nil, nil
	case "key":
		cmd, err := valueToKVMCommand(d.Value)
		if err != nil {
			return nil, err
		}
		return []string{cmd}, nil
	case "type":
		text := strings.TrimSpace(d.Value)
		if text == "" {
			return nil, fmt.Errorf("type action needs non-empty value")
		}
		if !isPrintableASCII(text) {
			return nil, fmt.Errorf("type text must be printable ASCII")
		}
		var cmds []string
		for i := 0; i < len(text); i += 50 {
			end := i + 50
			if end > len(text) {
				end = len(text)
			}
			cmds = append(cmds, "t"+text[i:end])
		}
		return cmds, nil
	default:
		return nil, fmt.Errorf("unsupported action %q", d.Action)
	}
}

func (p *PiClient) SendDecision(ctx context.Context, d *AgentDecision) error {
	cmds, err := decisionToKVMCommands(d)
	if err != nil {
		return err
	}
	if len(cmds) == 0 {
		return nil
	}
	return p.SendCommands(ctx, cmds...)
}

func (p *PiClient) HandleStream(frames *FrameBuffer, store *Store) gin.HandlerFunc {
	if !p.Enabled() {
		return handleMockVideoStream(frames)
	}
	return func(c *gin.Context) {
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, p.stream, nil)
		if err != nil {
			c.Status(http.StatusBadGateway)
			return
		}
		client := &http.Client{Timeout: 0}
		res, err := client.Do(req)
		if err != nil {
			store.SetPiConnected(false)
			c.Status(http.StatusBadGateway)
			return
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			store.SetPiConnected(false)
			c.Status(http.StatusBadGateway)
			return
		}
		store.SetPiConnected(true)

		ct := res.Header.Get("Content-Type")
		if ct == "" {
			ct = "multipart/x-mixed-replace; boundary=frame"
		}
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Connection", "close")
		c.Header("Content-Type", ct)
		c.Header("Pragma", "no-cache")
		c.Status(http.StatusOK)

		flusher, _ := c.Writer.(interface{ Flush() })
		if err := copyMJPEG(res.Body, c.Writer, frames, func() {
			if flusher != nil {
				flusher.Flush()
			}
		}); err != nil && c.Request.Context().Err() == nil {
			store.SetPiConnected(false)
		}
	}
}

func copyMJPEG(r io.Reader, w io.Writer, frames *FrameBuffer, flush func()) error {
	br := bufio.NewReader(r)
	for {
		length := 0
		for {
			line, err := br.ReadBytes('\n')
			if err != nil {
				return err
			}
			if _, err := w.Write(line); err != nil {
				return err
			}
			trimmed := strings.TrimSpace(string(line))
			if strings.HasPrefix(strings.ToLower(trimmed), "content-length:") {
				n := strings.TrimSpace(trimmed[len("content-length:"):])
				length, _ = strconv.Atoi(n)
			}
			if bytes.Equal(line, []byte("\r\n")) || bytes.Equal(line, []byte("\n")) {
				break
			}
		}
		if length <= 0 {
			if flush != nil {
				flush()
			}
			continue
		}
		jpeg := make([]byte, length)
		if _, err := io.ReadFull(br, jpeg); err != nil {
			return err
		}
		if _, err := w.Write(jpeg); err != nil {
			return err
		}
		frames.Put(jpeg)
		if flush != nil {
			flush()
		}
	}
}

func valueToKVMCommand(value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", fmt.Errorf("empty key")
	}
	upper := strings.ToUpper(v)
	special := map[string]string{
		"ENTER":     "kB0",
		"RETURN":    "kB0",
		"ESC":       "kB1",
		"ESCAPE":    "kB1",
		"BACKSPACE": "kB2",
		"TAB":       "kB3",
		"SPACE":     "k20",
		"UP":        "kDA",
		"DOWN":      "kD9",
		"LEFT":      "kD8",
		"RIGHT":     "kD7",
		"INSERT":    "kD1",
		"HOME":      "kD2",
		"PAGEUP":    "kD3",
		"DELETE":    "kD4",
		"END":       "kD5",
		"PAGEDOWN":  "kD6",
	}
	if cmd, ok := special[upper]; ok {
		return cmd, nil
	}
	if len(upper) >= 2 && upper[0] == 'F' {
		n, err := strconv.Atoi(upper[1:])
		if err == nil && n >= 1 && n <= 12 {
			return fmt.Sprintf("k%X", 0xC1+n), nil
		}
	}
	if len(v) == 1 {
		ch := v[0]
		if ch >= 0x20 && ch <= 0x7e {
			return fmt.Sprintf("k%X", ch), nil
		}
	}
	if len(v) <= 60 && isPrintableASCII(v) {
		return "t" + v, nil
	}
	return "", fmt.Errorf("cannot map key %q to kvmd command", v)
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return len(s) > 0
}
