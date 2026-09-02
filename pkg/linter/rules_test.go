package linter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

func TestRuleMCP001Description(t *testing.T) {
	rule := &RuleMCP001Description{}
	tok := tokenizer.NewFastTokenizer()

	longDesc := strings.Repeat("This is a very long description that goes on and on. ", 30)
	ctx := &StaticContext{
		Tools: []mcp.Tool{
			{Name: "bad_tool", Description: longDesc},
			{Name: "good_tool", Description: "Concise description"},
		},
		Tokenizer: tok,
	}

	findings := rule.CheckStatic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].RuleID != "MCP001" || findings[0].TargetName != "bad_tool" {
		t.Errorf("unexpected finding: %+v", findings[0])
	}
}

func TestRuleMCP002Monolithic(t *testing.T) {
	rule := &RuleMCP002Monolithic{}

	props := make(map[string]mcp.PropertySchema)
	for i := 0; i < 10; i++ {
		props[string(rune('a'+i))] = mcp.PropertySchema{Type: "string"}
	}

	ctx := &StaticContext{
		Tools: []mcp.Tool{
			{Name: "too_many_params", InputSchema: mcp.ToolInputSchema{Properties: props}},
			{Name: "normal_tool", InputSchema: mcp.ToolInputSchema{Properties: map[string]mcp.PropertySchema{"id": {Type: "string"}}}},
		},
	}

	findings := rule.CheckStatic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].RuleID != "MCP002" {
		t.Errorf("expected MCP002, got %s", findings[0].RuleID)
	}
}

func TestRuleMCP003Unbounded(t *testing.T) {
	rule := &RuleMCP003Unbounded{}

	ctx := &StaticContext{
		Tools: []mcp.Tool{
			{
				Name: "list_users",
				InputSchema: mcp.ToolInputSchema{
					Properties: map[string]mcp.PropertySchema{
						"filter": {Type: "string"},
					},
				},
			},
			{
				Name: "list_products",
				InputSchema: mcp.ToolInputSchema{
					Properties: map[string]mcp.PropertySchema{
						"limit":  {Type: "number"},
						"cursor": {Type: "string"},
					},
				},
			},
		},
	}

	findings := rule.CheckStatic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].TargetName != "list_users" {
		t.Errorf("expected list_users, got %s", findings[0].TargetName)
	}
}

func TestRuleMCP004FieldSelect(t *testing.T) {
	rule := &RuleMCP004FieldSelect{}

	ctx := &StaticContext{
		Tools: []mcp.Tool{
			{
				Name: "get_user",
				InputSchema: mcp.ToolInputSchema{
					Properties: map[string]mcp.PropertySchema{
						"user_id": {Type: "string"},
					},
				},
			},
			{
				Name: "get_order",
				InputSchema: mcp.ToolInputSchema{
					Properties: map[string]mcp.PropertySchema{
						"order_id": {Type: "string"},
						"fields":   {Type: "array"},
					},
				},
			},
		},
	}

	findings := rule.CheckStatic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].TargetName != "get_user" {
		t.Errorf("expected get_user, got %s", findings[0].TargetName)
	}
}

func TestRuleMCP005Metadata(t *testing.T) {
	rule := &RuleMCP005Metadata{}
	tok := tokenizer.NewFastTokenizer()

	wasteResult, _ := json.Marshal(map[string]any{
		"server_version":    "v1.0.4",
		"server_info":       "NodeJS MCP Server with verbose cluster debug metadata",
		"execution_time_ms": 124,
		"timestamp":         "2026-09-02T13:00:00Z",
		"content": []map[string]any{
			{"type": "text", "text": "{\"user_id\": 123}"},
		},
	})

	ctx := &DynamicContext{
		Tokenizer: tok,
		Traces: []TraceEvent{
			{
				Method:   "tools/call",
				ToolName: "fetch_data",
				Response: &mcp.Response{Result: wasteResult},
			},
			{
				Method:   "tools/call",
				ToolName: "fetch_data",
				Response: &mcp.Response{Result: wasteResult},
			},
		},
	}

	findings := rule.CheckDynamic(ctx)
	if len(findings) == 0 {
		t.Fatalf("expected finding for MCP005 metadata waste, got 0")
	}
}

func TestRuleMCP006JSONKeys(t *testing.T) {
	rule := &RuleMCP006JSONKeys{}

	ctx := &StaticContext{
		Tools: []mcp.Tool{
			{
				Name: "update_customer",
				InputSchema: mcp.ToolInputSchema{
					Properties: map[string]mcp.PropertySchema{
						"customer_account_first_name":    {Type: "string"},
						"customer_account_last_name":     {Type: "string"},
						"customer_account_email_address": {Type: "string"},
						"customer_account_phone_number":  {Type: "string"},
					},
				},
			},
		},
	}

	findings := rule.CheckStatic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for MCP006, got %d", len(findings))
	}
}

func TestRuleMCP007EnumBloat(t *testing.T) {
	rule := &RuleMCP007EnumBloat{}

	enums := make([]any, 40)
	for i := 0; i < 40; i++ {
		enums[i] = string(rune('A' + i))
	}

	ctx := &StaticContext{
		Tools: []mcp.Tool{
			{
				Name: "set_category",
				InputSchema: mcp.ToolInputSchema{
					Properties: map[string]mcp.PropertySchema{
						"category": {Type: "string", Enum: enums},
					},
				},
			},
		},
	}

	findings := rule.CheckStatic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for MCP007, got %d", len(findings))
	}
}

func TestRuleMCP008Duplicate(t *testing.T) {
	rule := &RuleMCP008Duplicate{}

	props := map[string]mcp.PropertySchema{
		"query": {Type: "string"},
		"page":  {Type: "number"},
	}

	ctx := &StaticContext{
		Tools: []mcp.Tool{
			{Name: "search_users", Description: "Search for users in the database", InputSchema: mcp.ToolInputSchema{Properties: props}},
			{Name: "find_users", Description: "Search for users in the database", InputSchema: mcp.ToolInputSchema{Properties: props}},
		},
	}

	findings := rule.CheckStatic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for MCP008, got %d", len(findings))
	}
}

func TestRuleMCP009Docstring(t *testing.T) {
	rule := &RuleMCP009Docstring{}

	ctx := &StaticContext{
		Tools: []mcp.Tool{
			{
				Name: "undocumented_tool",
				InputSchema: mcp.ToolInputSchema{
					Properties: map[string]mcp.PropertySchema{
						"param1": {Type: "string", Description: ""},
					},
				},
			},
		},
	}

	findings := rule.CheckStatic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for MCP009, got %d", len(findings))
	}
}

func TestRuleMCP010PayloadSize(t *testing.T) {
	rule := &RuleMCP010PayloadSize{}

	res, _ := json.Marshal(map[string]any{"content": "data"})
	ctx := &DynamicContext{
		Traces: []TraceEvent{
			{
				Method:     "tools/call",
				ToolName:   "heavy_tool",
				TokenCount: 12000,
				Response:   &mcp.Response{Result: res},
			},
		},
	}

	findings := rule.CheckDynamic(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for MCP010, got %d", len(findings))
	}
}
