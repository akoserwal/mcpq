package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

// SSEProxy acts as a reverse proxy for SSE MCP servers while recording traces.
type SSEProxy struct {
	targetURL *url.URL
	recorder  *Recorder
	server    *http.Server
}

// NewSSEProxy creates an SSEProxy pointing to a remote MCP SSE target.
func NewSSEProxy(targetURLStr string, tok tokenizer.Tokenizer) (*SSEProxy, error) {
	u, err := url.Parse(targetURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	if tok == nil {
		tok = tokenizer.DefaultTokenizer()
	}

	return &SSEProxy{
		targetURL: u,
		recorder:  NewRecorder(targetURLStr, tok),
	}, nil
}

// Recorder returns the attached trace recorder.
func (p *SSEProxy) Recorder() *Recorder {
	return p.recorder
}

// Start begins listening on listenAddr.
func (p *SSEProxy) Start(listenAddr string) error {
	p.server = &http.Server{Addr: listenAddr, Handler: p.handler()}
	return p.server.ListenAndServe()
}

func (p *SSEProxy) handler() http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(p.targetURL)
	proxy.ModifyResponse = func(resp *http.Response) error {
		mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if mediaType == "text/event-stream" || mediaType == "application/json" {
			resp.Body = &recordingBody{ReadCloser: resp.Body, recorder: p.recorder, sse: mediaType == "text/event-stream"}
		}
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If POST request with JSON-RPC body
		if r.Method == http.MethodPost {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				var req mcp.Request
				if err := json.Unmarshal(bodyBytes, &req); err == nil && req.JSONRPC != "" {
					p.recorder.RecordRequest(&req)
				}
			}
		}

		// Forward via reverse proxy
		r.Host = p.targetURL.Host
		proxy.ServeHTTP(w, r)
	})

	return mux
}

// Shutdown stops the proxy server.
func (p *SSEProxy) Shutdown(ctx context.Context) error {
	if p.server != nil {
		return p.server.Shutdown(ctx)
	}
	return nil
}
