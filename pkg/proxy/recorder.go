package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

// TraceFile represents the serialized trace file format.
type TraceFile struct {
	Version    string              `json:"version"`
	RecordedAt time.Time           `json:"recordedAt"`
	ServerCmd  string              `json:"serverCmd,omitempty"`
	Events     []linter.TraceEvent `json:"events"`
}

// Recorder records JSON-RPC requests and responses thread-safely.
type Recorder struct {
	mu        sync.Mutex
	events    []linter.TraceEvent
	pending   map[int64]pendingReq
	tok       tokenizer.Tokenizer
	serverCmd string
}

type pendingReq struct {
	req       *mcp.Request
	toolName  string
	startTime time.Time
}

// NewRecorder creates a new Recorder instance.
func NewRecorder(serverCmd string, tok tokenizer.Tokenizer) *Recorder {
	if tok == nil {
		tok = tokenizer.DefaultTokenizer()
	}
	return &Recorder{
		events:    make([]linter.TraceEvent, 0),
		pending:   make(map[int64]pendingReq),
		tok:       tok,
		serverCmd: serverCmd,
	}
}

// RecordRequest records an outbound/inbound JSON-RPC request.
func (r *Recorder) RecordRequest(req *mcp.Request) {
	if req == nil {
		return
	}

	toolName := ""
	if req.Method == "tools/call" && len(req.Params) > 0 {
		var params mcp.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			toolName = params.Name
		}
	}

	rawBytes, _ := json.Marshal(req)
	tokCount := r.tok.CountTokens(string(rawBytes))

	event := linter.TraceEvent{
		Timestamp:  time.Now(),
		Direction:  "request",
		Method:     req.Method,
		ToolName:   toolName,
		Request:    req,
		TokenCount: tokCount,
		RawBytes:   len(rawBytes),
	}

	r.mu.Lock()
	r.events = append(r.events, event)
	if req.ID != nil {
		if id, ok := extractIntID(req.ID); ok {
			r.pending[id] = pendingReq{
				req:       req,
				toolName:  toolName,
				startTime: time.Now(),
			}
		}
	}
	r.mu.Unlock()
}

// RecordResponse records a response and correlates it with pending request.
func (r *Recorder) RecordResponse(resp *mcp.Response) {
	if resp == nil {
		return
	}

	rawBytes, _ := json.Marshal(resp)
	tokCount := r.tok.CountTokens(string(rawBytes))

	var toolName string
	var method string
	var durationMs int64

	r.mu.Lock()
	if resp.ID != nil {
		if id, ok := extractIntID(resp.ID); ok {
			if p, found := r.pending[id]; found {
				delete(r.pending, id)
				toolName = p.toolName
				method = p.req.Method
				durationMs = time.Since(p.startTime).Milliseconds()
			}
		}
	}

	event := linter.TraceEvent{
		Timestamp:  time.Now(),
		Direction:  "response",
		Method:     method,
		ToolName:   toolName,
		Response:   resp,
		DurationMs: durationMs,
		TokenCount: tokCount,
		RawBytes:   len(rawBytes),
	}
	r.events = append(r.events, event)
	r.mu.Unlock()
}

// Events returns a slice of all recorded trace events.
func (r *Recorder) Events() []linter.TraceEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make([]linter.TraceEvent, len(r.events))
	copy(copied, r.events)
	return copied
}

// SaveToFile writes recorded traces to the specified path.
func (r *Recorder) SaveToFile(filePath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tf := TraceFile{
		Version:    "1.0",
		RecordedAt: time.Now(),
		ServerCmd:  r.serverCmd,
		Events:     r.events,
	}

	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode trace file: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// LoadTraces reads a trace file from disk.
func LoadTraces(filePath string) ([]linter.TraceEvent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace file %q: %w", filePath, err)
	}

	// Try loading as TraceFile
	var tf TraceFile
	if err := json.Unmarshal(data, &tf); err == nil && len(tf.Events) > 0 {
		return tf.Events, nil
	}

	// Fallback: try loading directly as []linter.TraceEvent
	var events []linter.TraceEvent
	if err := json.Unmarshal(data, &events); err == nil {
		return events, nil
	}

	return nil, fmt.Errorf("file %q is not a valid MCPQ trace JSON file", filePath)
}

func extractIntID(idVal any) (int64, bool) {
	switch v := idVal.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	}
	return 0, false
}
