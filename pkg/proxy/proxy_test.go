package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

func TestRecorderSaveAndLoad(t *testing.T) {
	tok := tokenizer.NewFastTokenizer()
	rec := NewRecorder("test_cmd", tok)

	req := &mcp.Request{
		JSONRPC: "2.0",
		ID:      int64(1),
		Method:  "tools/list",
	}
	rec.RecordRequest(req)

	resBytes, _ := json.Marshal(map[string]any{"tools": []string{}})
	resp := &mcp.Response{
		JSONRPC: "2.0",
		ID:      int64(1),
		Result:  resBytes,
	}
	rec.RecordResponse(resp)

	events := rec.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	tmpDir := t.TempDir()
	tracePath := filepath.Join(tmpDir, "test_trace.json")

	if err := rec.SaveToFile(tracePath); err != nil {
		t.Fatalf("failed to save traces: %v", err)
	}

	if _, err := os.Stat(tracePath); os.IsNotExist(err) {
		t.Fatalf("trace file was not created on disk")
	}

	loaded, err := LoadTraces(tracePath)
	if err != nil {
		t.Fatalf("failed to load traces: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 loaded events, got %d", len(loaded))
	}
}
