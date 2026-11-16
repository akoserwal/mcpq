package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SSEClient implements the MCP Client interface over Server-Sent Events / HTTP.
type SSEClient struct {
	baseURL    *url.URL
	messageURL *url.URL
	httpClient *http.Client

	mu      sync.Mutex
	pending map[int64]chan *Response
	reqID   atomic.Int64
	closed  bool
	cancel  context.CancelFunc
}

// NewSSEClient creates an SSEClient connected to the provided endpoint URL.
func NewSSEClient(serverURL string) (*SSEClient, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 0, // No timeout for SSE connection
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &SSEClient{
		baseURL:    u,
		httpClient: httpClient,
		pending:    make(map[int64]chan *Response),
		cancel:     cancel,
	}

	endpointReady := make(chan error, 1)
	go c.listenSSE(ctx, endpointReady)

	select {
	case err := <-endpointReady:
		if err != nil {
			cancel()
			return nil, err
		}
	case <-time.After(15 * time.Second):
		cancel()
		return nil, fmt.Errorf("timeout waiting for SSE endpoint initialization")
	}

	return c, nil
}

func (c *SSEClient) listenSSE(ctx context.Context, ready chan<- error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL.String(), nil)
	if err != nil {
		ready <- fmt.Errorf("failed to create SSE request: %w", err)
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		ready <- fmt.Errorf("failed to connect to SSE stream: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ready <- fmt.Errorf("SSE request returned status: %s", resp.Status)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string
	var currentData bytes.Buffer
	notifiedReady := false

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// Dispatch event
			eventData := currentData.String()
			if currentEvent == "endpoint" {
				relURL, err := url.Parse(strings.TrimSpace(eventData))
				if err == nil {
					c.mu.Lock()
					c.messageURL = c.baseURL.ResolveReference(relURL)
					c.mu.Unlock()
					if !notifiedReady {
						notifiedReady = true
						ready <- nil
					}
				}
			} else if currentEvent == "message" || currentEvent == "" {
				var r Response
				if err := json.Unmarshal([]byte(eventData), &r); err == nil && r.ID != nil {
					var id int64
					switch v := r.ID.(type) {
					case float64:
						id = int64(v)
					case int64:
						id = v
					case int:
						id = int64(v)
					}
					c.mu.Lock()
					ch, ok := c.pending[id]
					if ok {
						delete(c.pending, id)
					}
					c.mu.Unlock()
					if ok {
						ch <- &r
					}
				}
			}

			currentEvent = ""
			currentData.Reset()
			continue
		}

		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			if strings.HasPrefix(data, " ") {
				data = data[1:]
			}
			currentData.WriteString(data)
			currentData.WriteString("\n")
		}
	}

	if !notifiedReady {
		ready <- fmt.Errorf("SSE connection closed before receiving endpoint event")
	}

	c.mu.Lock()
	c.closed = true
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.mu.Unlock()
}

// SendRequest sends a JSON-RPC request over HTTP POST.
func (c *SSEClient) SendRequest(ctx context.Context, method string, params any) (*Response, error) {
	id := c.reqID.Add(1)

	var rawParams json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		rawParams = data
	}

	reqPayload := Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}

	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client connection is closed")
	}
	msgURL := c.messageURL
	if msgURL == nil {
		msgURL = c.baseURL
	}
	respCh := make(chan *Response, 1)
	c.pending[id] = respCh
	c.mu.Unlock()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", msgURL.String(), bytes.NewReader(reqBytes))
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	postClient := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := postClient.Do(httpReq)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("HTTP post failed: %w", err)
	}
	defer httpResp.Body.Close()

	// If response is returned directly in HTTP response body (standard JSON-RPC over HTTP)
	if httpResp.Header.Get("Content-Type") == "application/json" {
		body, err := io.ReadAll(httpResp.Body)
		if err == nil && len(body) > 0 {
			var resp Response
			if err := json.Unmarshal(body, &resp); err == nil && resp.JSONRPC != "" {
				c.mu.Lock()
				delete(c.pending, id)
				c.mu.Unlock()
				if resp.Error != nil {
					return nil, fmt.Errorf("JSON-RPC error (%d): %s", resp.Error.Code, resp.Error.Message)
				}
				return &resp, nil
			}
		}
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case resp, ok := <-respCh:
		if !ok || resp == nil {
			return nil, fmt.Errorf("server closed connection without responding")
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("JSON-RPC error (%d): %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	}
}

// Initialize performs the MCP handshake over SSE.
func (c *SSEClient) Initialize(ctx context.Context) (*InitializeResult, error) {
	params := InitializeParams{
		ProtocolVersion: LatestProtocolVersion,
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{ListChanged: true},
		},
		ClientInfo: Implementation{
			Name:    "mcpq",
			Version: "0.1.0",
		},
	}

	resp, err := c.SendRequest(ctx, "initialize", params)
	if err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode initialize result: %w", err)
	}

	notification, err := json.Marshal(Request{JSONRPC: JSONRPCVersion, Method: "notifications/initialized"})
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	endpoint := c.messageURL.String()
	c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(notification))
	if err != nil {
		return nil, fmt.Errorf("create initialized notification: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send initialized notification: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("initialized notification returned status: %s", response.Status)
	}
	return &result, nil
}

// ListTools lists all available tools from the server.
func (c *SSEClient) ListTools(ctx context.Context) ([]Tool, error) {
	var allTools []Tool
	var cursor *string

	for {
		var params map[string]any
		if cursor != nil {
			params = map[string]any{"cursor": *cursor}
		}

		resp, err := c.SendRequest(ctx, "tools/list", params)
		if err != nil {
			return nil, fmt.Errorf("tools/list failed: %w", err)
		}

		var result ListToolsResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("failed to decode tools/list result: %w", err)
		}

		allTools = append(allTools, result.Tools...)
		if result.NextCursor == nil || *result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	return allTools, nil
}

// ListResources lists all available resources.
func (c *SSEClient) ListResources(ctx context.Context) ([]Resource, error) {
	var allResources []Resource
	var cursor *string

	for {
		var params map[string]any
		if cursor != nil {
			params = map[string]any{"cursor": *cursor}
		}

		resp, err := c.SendRequest(ctx, "resources/list", params)
		if err != nil {
			return nil, fmt.Errorf("resources/list failed: %w", err)
		}

		var result ListResourcesResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("failed to decode resources/list result: %w", err)
		}

		allResources = append(allResources, result.Resources...)
		if result.NextCursor == nil || *result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	return allResources, nil
}

// ListPrompts lists all available prompts.
func (c *SSEClient) ListPrompts(ctx context.Context) ([]Prompt, error) {
	var allPrompts []Prompt
	var cursor *string

	for {
		var params map[string]any
		if cursor != nil {
			params = map[string]any{"cursor": *cursor}
		}

		resp, err := c.SendRequest(ctx, "prompts/list", params)
		if err != nil {
			return nil, fmt.Errorf("prompts/list failed: %w", err)
		}

		var result ListPromptsResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("failed to decode prompts/list result: %w", err)
		}

		allPrompts = append(allPrompts, result.Prompts...)
		if result.NextCursor == nil || *result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	return allPrompts, nil
}

// CallTool executes a specific tool on the MCP server.
func (c *SSEClient) CallTool(ctx context.Context, name string, args map[string]any) (*CallToolResult, error) {
	if args == nil {
		args = make(map[string]any)
	}

	params := CallToolParams{
		Name:      name,
		Arguments: args,
	}

	resp, err := c.SendRequest(ctx, "tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("tools/call %q failed: %w", name, err)
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode tools/call result: %w", err)
	}

	return &result, nil
}

// Close terminates the SSE client.
func (c *SSEClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	c.cancel()
	return nil
}
