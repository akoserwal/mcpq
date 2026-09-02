package mcp

import (
	"context"
)

// Client defines the common interface for interacting with an MCP server.
type Client interface {
	// Initialize performs the MCP handshake.
	Initialize(ctx context.Context) (*InitializeResult, error)

	// ListTools lists all available tools.
	ListTools(ctx context.Context) ([]Tool, error)

	// ListResources lists all available resources.
	ListResources(ctx context.Context) ([]Resource, error)

	// ListPrompts lists all available prompts.
	ListPrompts(ctx context.Context) ([]Prompt, error)

	// CallTool executes a specific tool with arguments.
	CallTool(ctx context.Context, name string, args map[string]any) (*CallToolResult, error)

	// Close terminates the transport connection.
	Close() error
}
