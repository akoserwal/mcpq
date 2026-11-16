package mcp

import (
	"encoding/json"
)

// JSON-RPC 2.0 protocol constants
const (
	JSONRPCVersion        = "2.0"
	LatestProtocolVersion = "2024-11-05"
)

// Request is a standard JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a standard JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a standard JSON-RPC error.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return e.Message
}

// InitializeParams defines client info and capabilities sent on initialization.
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// ClientCapabilities represents MCP client capabilities.
type ClientCapabilities struct {
	Roots        *RootsCapability    `json:"roots,omitempty"`
	Sampling     *SamplingCapability `json:"sampling,omitempty"`
	Experimental map[string]any      `json:"experimental,omitempty"`
}

// RootsCapability defines roots capability options.
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability defines sampling options.
type SamplingCapability struct{}

// Implementation represents server or client info.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult represents server response to initialize.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

// ServerCapabilities represents MCP server capabilities.
type ServerCapabilities struct {
	Tools        *ToolsCapability     `json:"tools,omitempty"`
	Resources    *ResourcesCapability `json:"resources,omitempty"`
	Prompts      *PromptsCapability   `json:"prompts,omitempty"`
	Logging      *LoggingCapability   `json:"logging,omitempty"`
	Experimental map[string]any       `json:"experimental,omitempty"`
}

// ToolsCapability defines tool options.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability defines resource options.
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability defines prompt options.
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability defines logging capability.
type LoggingCapability struct{}

// Tool represents an MCP tool definition.
type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	InputSchema  ToolInputSchema `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
	RawJSON      json.RawMessage `json:"-"`
}

// ToolInputSchema represents the JSON Schema for a tool's input arguments.
type ToolInputSchema struct {
	Type                 string                    `json:"type"`
	Properties           map[string]PropertySchema `json:"properties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AdditionalProperties any                       `json:"additionalProperties,omitempty"`
	Description          string                    `json:"description,omitempty"`
}

// PropertySchema represents the schema of an individual tool parameter.
type PropertySchema struct {
	Type        string                    `json:"type,omitempty"`
	Description string                    `json:"description,omitempty"`
	Enum        []any                     `json:"enum,omitempty"`
	Items       *PropertySchema           `json:"items,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
	Default     any                       `json:"default,omitempty"`
}

// ListToolsResult represents the result of tools/list.
type ListToolsResult struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// Resource represents an MCP resource definition.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

// ListResourcesResult represents the result of resources/list.
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor *string    `json:"nextCursor,omitempty"`
}

// Prompt represents an MCP prompt template.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes an argument for a prompt template.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ListPromptsResult represents the result of prompts/list.
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor *string  `json:"nextCursor,omitempty"`
}

// CallToolParams represents parameters for tools/call.
type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// CallToolResult represents the response from tools/call.
type CallToolResult struct {
	Content           []Content       `json:"content"`
	IsError           bool            `json:"isError,omitempty"`
	StructuredContent json.RawMessage `json:"structuredContent,omitempty"`
	RawJSON           json.RawMessage `json:"-"`
}

// Content represents content returned by tool execution.
type Content struct {
	Type     string `json:"type"` // "text", "image", "resource"
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
}

// UnmarshalJSON retains the complete received definition for token accounting,
// including JSON Schema keywords and MCP extensions unknown to the typed view.
func (t *Tool) UnmarshalJSON(data []byte) error {
	type tool Tool
	var decoded tool
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*t = Tool(decoded)
	t.RawJSON = append(json.RawMessage(nil), data...)
	return nil
}

// UnmarshalJSON retains the complete received result for benchmark accounting.
func (r *CallToolResult) UnmarshalJSON(data []byte) error {
	type result CallToolResult
	var decoded result
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = CallToolResult(decoded)
	r.RawJSON = append(json.RawMessage(nil), data...)
	return nil
}
