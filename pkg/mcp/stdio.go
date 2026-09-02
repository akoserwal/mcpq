package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// StdioClient implements the MCP Client interface over standard input/output.
type StdioClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	scanner *bufio.Scanner

	mu       sync.Mutex
	pending  map[int64]chan *Response
	reqID    atomic.Int64
	closed   bool
	closeCh  chan struct{}
}

// NewStdioClient starts the target command and creates a StdioClient.
func NewStdioClient(command string, args []string, env []string) (*StdioClient, error) {
	cmd := exec.Command(command, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to open stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start command %q: %w", command, err)
	}

	c := &StdioClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		scanner: bufio.NewScanner(stdout),
		pending: make(map[int64]chan *Response),
		closeCh: make(chan struct{}),
	}

	// Set large scanner buffer for big tool payloads (e.g. up to 16MB)
	const maxScanBuffer = 16 * 1024 * 1024
	buf := make([]byte, 64*1024)
	c.scanner.Buffer(buf, maxScanBuffer)

	go c.readLoop()

	return c, nil
}

func (c *StdioClient) readLoop() {
	for c.scanner.Scan() {
		line := c.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		if resp.ID != nil {
			var id int64
			switch v := resp.ID.(type) {
			case float64:
				id = int64(v)
			case int64:
				id = v
			case int:
				id = int64(v)
			default:
				continue
			}

			c.mu.Lock()
			ch, ok := c.pending[id]
			if ok {
				delete(c.pending, id)
			}
			c.mu.Unlock()

			if ok {
				ch <- &resp
			}
		}
	}

	c.mu.Lock()
	c.closed = true
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.mu.Unlock()
}

// SendRequest sends a JSON-RPC request and waits for the response.
func (c *StdioClient) SendRequest(ctx context.Context, method string, params any) (*Response, error) {
	id := c.reqID.Add(1)

	var rawParams json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		rawParams = data
	}

	req := Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	respCh := make(chan *Response, 1)
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client connection is closed")
	}
	c.pending[id] = respCh
	c.mu.Unlock()

	c.mu.Lock()
	_, writeErr := c.stdin.Write(reqBytes)
	c.mu.Unlock()

	if writeErr != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to write to stdin: %w", writeErr)
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

// SendNotification sends a one-way notification without expecting a response.
func (c *StdioClient) SendNotification(method string, params any) error {
	var rawParams json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
		rawParams = data
	}

	req := Request{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  rawParams,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("client connection is closed")
	}

	_, writeErr := c.stdin.Write(reqBytes)
	return writeErr
}

// Initialize performs the MCP handshake.
func (c *StdioClient) Initialize(ctx context.Context) (*InitializeResult, error) {
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
		return nil, fmt.Errorf("initialize request failed: %w", err)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to decode initialize result: %w", err)
	}

	// Send initialized notification as per MCP spec
	_ = c.SendNotification("notifications/initialized", map[string]any{})

	return &result, nil
}

// ListTools lists all available tools from the server.
func (c *StdioClient) ListTools(ctx context.Context) ([]Tool, error) {
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
func (c *StdioClient) ListResources(ctx context.Context) ([]Resource, error) {
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
func (c *StdioClient) ListPrompts(ctx context.Context) ([]Prompt, error) {
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
func (c *StdioClient) CallTool(ctx context.Context, name string, args map[string]any) (*CallToolResult, error) {
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

// Close terminates the subprocess and closes all streams.
func (c *StdioClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
	return nil
}
