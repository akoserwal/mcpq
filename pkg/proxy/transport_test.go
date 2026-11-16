package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

func TestStdioReturnsWhenServerExits(t *testing.T) {
	in, writer := io.Pipe()
	defer in.Close()
	defer writer.Close()
	p := NewStdioProxy("sh", []string{"-c", `printf '%s\n' '{"jsonrpc":"2.0","id":1,"result":{}}'`}, tokenizer.NewFastTokenizer())
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- p.Run(in, &out, io.Discard) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(out.Bytes(), []byte(`"result":{}`)) {
			t.Fatalf("lost final response: %s", out.Bytes())
		}
		if len(p.Recorder().Events()) != 1 {
			t.Fatal("final response not recorded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("proxy blocked on open client stdin")
	}
}

func TestSSEProxyRecordsResponses(t *testing.T) {
	for _, tc := range []struct{ name, contentType, body string }{
		{"sse", "text/event-stream; charset=utf-8", "event: endpoint\r\ndata: /messages\r\n\r\nevent: message\r\ndata: {\r\ndata: \"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"content\":[]}}\r\n\r\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/test\"}\n\n"},
		{"json", "application/json; charset=utf-8", `{"jsonrpc":"2.0","id":1,"result":{"content":[]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				for _, c := range []byte(tc.body) {
					w.Write([]byte{c})
					w.(http.Flusher).Flush()
				}
			}))
			defer upstream.Close()
			p, err := NewSSEProxy(upstream.URL, tokenizer.NewFastTokenizer())
			if err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(p.handler())
			defer srv.Close()
			resp, err := http.Post(srv.URL, "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test"}}`))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			got, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.body {
				t.Fatal("proxy changed response bytes")
			}
			events := p.Recorder().Events()
			if len(events) != 2 || events[1].Response == nil || events[1].ToolName != "test" || events[1].TokenCount == 0 {
				t.Fatalf("invalid recorded events: %+v", events)
			}
		})
	}
}
