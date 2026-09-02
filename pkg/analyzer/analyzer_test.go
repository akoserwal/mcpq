package analyzer

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

func TestAnalyzeSchema(t *testing.T) {
	tok := tokenizer.NewFastTokenizer()
	tools := []mcp.Tool{
		{
			Name:        "get_product",
			Description: "Get product details",
			InputSchema: mcp.ToolInputSchema{
				Properties: map[string]mcp.PropertySchema{
					"id": {Type: "string", Description: "Product ID"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "list_products",
			Description: "List products with limit",
			InputSchema: mcp.ToolInputSchema{
				Properties: map[string]mcp.PropertySchema{
					"limit":  {Type: "number"},
					"cursor": {Type: "string"},
				},
			},
		},
	}

	analysis := AnalyzeSchema(tools, nil, nil, tok)
	if analysis.ToolCount != 2 {
		t.Errorf("expected 2 tools, got %d", analysis.ToolCount)
	}
	if analysis.TotalParams != 3 {
		t.Errorf("expected 3 total params, got %d", analysis.TotalParams)
	}
	if analysis.AvgParamsPerTool != 1.5 {
		t.Errorf("expected avg 1.5 params, got %f", analysis.AvgParamsPerTool)
	}
}

func TestAnalyzeTraces(t *testing.T) {
	tok := tokenizer.NewFastTokenizer()

	res1, _ := json.Marshal(map[string]any{"status": "ok"})
	res2, _ := json.Marshal(map[string]any{"status": "ok", "items": []string{"a", "b", "c"}})

	events := []linter.TraceEvent{
		{
			Timestamp:  time.Now(),
			Direction:  "response",
			Method:     "tools/call",
			ToolName:   "ping",
			TokenCount: 50,
			DurationMs: 12,
			Response:   &mcp.Response{Result: res1},
		},
		{
			Timestamp:  time.Now(),
			Direction:  "response",
			Method:     "tools/call",
			ToolName:   "list",
			TokenCount: 300,
			DurationMs: 45,
			Response:   &mcp.Response{Result: res2},
		},
	}

	analysis := AnalyzeTraces(events, tok)
	if analysis.TotalToolCalls != 2 {
		t.Errorf("expected 2 calls, got %d", analysis.TotalToolCalls)
	}
	if analysis.P50Tokens != 50 && analysis.P50Tokens != 300 {
		t.Errorf("unexpected P50: %d", analysis.P50Tokens)
	}
}
