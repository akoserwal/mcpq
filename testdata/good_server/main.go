package main

import (
	"bufio"
	"encoding/json"
	"os"
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
					"name":    "good-server",
					"version": "1.0.0",
				},
			})
		case "notifications/initialized":
			// no-op
		case "tools/list":
			sendResp(req.ID, map[string]any{
				"tools": getGoodTools(),
			})
		case "tools/call":
			sendResp(req.ID, map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": `{"items":[{"id":"item_1","name":"Widget A"}],"nextCursor":"cur_2"}`,
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

func getGoodTools() []map[string]any {
	return []map[string]any{
		{
			"name":        "list_items",
			"description": "List items with cursor-based pagination.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit":  map[string]any{"type": "number", "description": "Maximum number of items to return (default 20)"},
					"cursor": map[string]any{"type": "string", "description": "Pagination cursor from previous page"},
				},
			},
		},
		{
			"name":        "get_item",
			"description": "Fetch item details with optional field filtering.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string", "description": "Item identifier"},
					"fields": map[string]any{"type": "array", "description": "List of specific field attributes to return"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "create_item",
			"description": "Create a new item in the workspace.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":        map[string]any{"type": "string", "description": "Name of the item"},
					"description": map[string]any{"type": "string", "description": "Optional item summary"},
				},
				"required": []string{"name"},
			},
		},
	}
}
