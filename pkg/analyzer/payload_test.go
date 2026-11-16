package analyzer

import (
	"encoding/json"
	"testing"

	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

func TestSchemaCountsCompleteReceivedDefinition(t *testing.T) {
	payload := `{"name":"convert","inputSchema":{"type":"object","title":"Arguments","properties":{"uri":{"title":"Uri","type":"string"}}},"outputSchema":{"type":"object","properties":{"result":{"type":"string"}}},"annotations":{"readOnlyHint":true}}`
	var tool mcp.Tool
	if err := json.Unmarshal([]byte(payload), &tool); err != nil {
		t.Fatal(err)
	}
	tok := tokenizer.NewFastTokenizer()
	result := AnalyzeSchema([]mcp.Tool{tool}, nil, nil, tok)
	if result.TotalTokens != tok.CountTokens(payload) {
		t.Fatalf("schema tokens=%d want %d", result.TotalTokens, tok.CountTokens(payload))
	}
}
