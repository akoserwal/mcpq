package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			sendResp(req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "bad-server",
					"version": "1.0.0",
				},
			})
		case "notifications/initialized":
			// no-op
		case "tools/list":
			sendResp(req.ID, map[string]any{
				"tools": getBadTools(),
			})
		case "tools/call":
			// Returns payload with repeated metadata waste
			sendResp(req.ID, map[string]any{
				"server_version":    "1.0.0-build-9872",
				"execution_time_ms": 120,
				"timestamp":         "2026-09-02T13:00:00Z",
				"content": []map[string]any{
					{
						"type": "text",
						"text": `{"user_account_first_name":"John","user_account_last_name":"Doe","user_account_email_address":"john@example.com"}`,
					},
				},
			})
		case "resources/list":
			sendResp(req.ID, map[string]any{"resources": []any{}})
		case "prompts/list":
			sendResp(req.ID, map[string]any{"prompts": []any{}})
		default:
			if req.ID != nil {
				sendResp(req.ID, map[string]any{})
			}
		}
	}
}

func sendResp(id any, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(append(data, '\n'))
}

func getBadTools() []map[string]any {
	giantEnum := make([]string, 35)
	for i := 0; i < 35; i++ {
		giantEnum[i] = fmt.Sprintf("STATUS_CODE_ENUM_ENTRY_%d", i)
	}

	hugeDescription := strings.Repeat("This tool does everything and requires reading our entire manual. Please note section 4.2 of RFC 9999. ", 20)

	return []map[string]any{
		{
			"name":        "huge_docstring_tool",
			"description": hugeDescription,
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"input": map[string]any{"type": "string", "description": "some input"},
				},
			},
		},
		{
			"name":        "manage_everything_mega_tool",
			"description": "Monolithic dispatcher tool",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action":    map[string]any{"type": "string", "enum": []string{"create", "read", "update", "delete", "archive", "export"}, "description": "action to perform"},
					"param1":    map[string]any{"type": "string"},
					"param2":    map[string]any{"type": "string"},
					"param3":    map[string]any{"type": "string"},
					"param4":    map[string]any{"type": "string"},
					"param5":    map[string]any{"type": "string"},
					"param6":    map[string]any{"type": "string"},
					"param7":    map[string]any{"type": "string"},
					"param8":    map[string]any{"type": "string"},
					"param9":    map[string]any{"type": "string"},
					"param10":   map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "list_all_records_unbounded",
			"description": "Lists all records in the database without pagination",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filter_query": map[string]any{"type": "string", "description": "optional text query"},
				},
			},
		},
		{
			"name":        "get_customer_details",
			"description": "Fetches complete customer object without field selection",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"customer_account_first_name":    map[string]any{"type": "string"},
					"customer_account_last_name":     map[string]any{"type": "string"},
					"customer_account_email_address": map[string]any{"type": "string"},
					"customer_account_phone_number":  map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "select_status_bloat",
			"description": "Tool containing huge enum inline",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "string", "enum": giantEnum, "description": "Enum status"},
				},
			},
		},
		{
			"name":        "search_records",
			"description": "Search records by term",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "find_records",
			"description": "Search records by term",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
			},
		},
	}
}
