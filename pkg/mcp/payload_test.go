package mcp

import (
	"encoding/json"
	"testing"
)

func TestPreserveToolAndResultPayloads(t *testing.T) {
	toolJSON := `{"name":"convert","description":"Convert document","inputSchema":{"type":"object","title":"Arguments","properties":{"uri":{"type":"string","title":"Uri"}}},"outputSchema":{"type":"object"},"annotations":{"readOnlyHint":true}}`
	var tool Tool
	if err := json.Unmarshal([]byte(toolJSON), &tool); err != nil {
		t.Fatal(err)
	}
	if string(tool.RawJSON) != toolJSON || len(tool.OutputSchema) == 0 {
		t.Fatal("tool schema fields lost")
	}
	resultJSON := `{"content":[{"type":"text","text":"hello"}],"isError":false,"structuredContent":{"result":"hello"},"_meta":{"extra":"preserve"}}`
	var result CallToolResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatal(err)
	}
	if string(result.RawJSON) != resultJSON || len(result.StructuredContent) == 0 {
		t.Fatal("result fields lost")
	}
}
