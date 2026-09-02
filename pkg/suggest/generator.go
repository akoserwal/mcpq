package suggest

import (
	"fmt"
	"strings"

	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/mcp"
)

// GenerateSuggestions analyzes tools and findings, generating progressive disclosure code suggestions.
func GenerateSuggestions(tools []mcp.Tool, findings []linter.Finding) []Suggestion {
	var suggestions []Suggestion
	toolMap := make(map[string]mcp.Tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}

	for _, f := range findings {
		tool, hasTool := toolMap[f.TargetName]

		switch f.RuleID {
		case "MCP003": // Unbounded response / pagination
			suggestions = append(suggestions, generatePaginationSuggestion(f.TargetName, tool, hasTool))
		case "MCP004": // No field selection
			suggestions = append(suggestions, generateFieldSelectSuggestion(f.TargetName, tool, hasTool))
		case "MCP002": // Monolithic tool
			suggestions = append(suggestions, generateDecomposeSuggestion(f.TargetName, tool, hasTool))
		case "MCP007": // Enum explosion
			suggestions = append(suggestions, generateEnumResourceSuggestion(f.TargetName, tool, hasTool))
		}
	}

	return suggestions
}

func generatePaginationSuggestion(toolName string, tool mcp.Tool, hasTool bool) Suggestion {
	tsCode := fmt.Sprintf(`// Optimized MCP Tool with Cursor Pagination
server.tool(
  "%s",
  "Fetch items with cursor pagination to avoid flooding LLM context",
  {
    limit: z.number().min(1).max(50).default(20).describe("Maximum items to return (default 20)"),
    cursor: z.string().optional().describe("Opaque pagination cursor from previous response"),
  },
  async ({ limit, cursor }) => {
    const { items, nextCursor } = await fetchPaginatedData(cursor, limit);
    return {
      content: [{
        type: "text",
        text: JSON.stringify({ items, nextCursor, hasMore: !!nextCursor })
      }]
    };
  }
);`, toolName)

	goCode := fmt.Sprintf(`// Optimized Go MCP Tool with Cursor Pagination
type %sArgs struct {
	Limit  int     `+"`json:\"limit,omitempty\"`"+`  // default: 20
	Cursor *string `+"`json:\"cursor,omitempty\"`"+` // pagination cursor
}

func handle%s(ctx context.Context, args %sArgs) (*mcp.CallToolResult, error) {
	if args.Limit <= 0 || args.Limit > 50 {
		args.Limit = 20
	}
	items, nextCursor, err := db.FetchPage(ctx, args.Cursor, args.Limit)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{Type: "text", Text: mustJSON(map[string]any{"items": items, "nextCursor": nextCursor})},
		},
	}, nil
}`, toPascalCase(toolName), toPascalCase(toolName), toPascalCase(toolName))

	return Suggestion{
		ToolName:        toolName,
		Pattern:         PatternPagination,
		Title:           fmt.Sprintf("Add Cursor Pagination to `%s`", toolName),
		Description:     "Enforce bounded collection sizes with limit (default 20) and cursor tokens to prevent exhausting LLM context windows.",
		EstimatedSaving: "60% - 90% token reduction per list query",
		TypeScriptCode:  tsCode,
		GoCode:          goCode,
	}
}

func generateFieldSelectSuggestion(toolName string, tool mcp.Tool, hasTool bool) Suggestion {
	tsCode := fmt.Sprintf(`// Optimized MCP Tool with Field Projection
server.tool(
  "%s",
  "Fetch entity details with optional field filtering",
  {
    id: z.string().describe("Entity ID to retrieve"),
    fields: z.array(z.string()).optional().describe("Optional list of specific fields to return (e.g. ['name', 'status'])"),
  },
  async ({ id, fields }) => {
    const record = await fetchEntity(id);
    const projected = fields && fields.length > 0 
      ? pickFields(record, fields) 
      : record;
    return {
      content: [{ type: "text", text: JSON.stringify(projected) }]
    };
  }
);`, toolName)

	goCode := fmt.Sprintf(`// Optimized Go MCP Tool with Field Projection
type %sArgs struct {
	ID     string   `+"`json:\"id\"`"+`
	Fields []string `+"`json:\"fields,omitempty\"`"+` // Optional projection
}

func handle%s(ctx context.Context, args %sArgs) (*mcp.CallToolResult, error) {
	record, err := db.Get(ctx, args.ID)
	if err != nil {
		return nil, err
	}
	filtered := filterFields(record, args.Fields)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{Type: "text", Text: mustJSON(filtered)},
		},
	}, nil
}`, toPascalCase(toolName), toPascalCase(toolName), toPascalCase(toolName))

	return Suggestion{
		ToolName:        toolName,
		Pattern:         PatternFieldSelect,
		Title:           fmt.Sprintf("Add Field Projection to `%s`", toolName),
		Description:     "Allow LLM agents to request only necessary attributes using a `fields` filter, eliminating unneeded payload weight.",
		EstimatedSaving: "40% - 80% token reduction on detail queries",
		TypeScriptCode:  tsCode,
		GoCode:          goCode,
	}
}

func generateDecomposeSuggestion(toolName string, tool mcp.Tool, hasTool bool) Suggestion {
	tsCode := `// Split monolithic dispatcher into discrete, focused tools
server.tool("create_item", "Create a new item", { name: z.string(), value: z.number() }, async (args) => { ... });
server.tool("update_item", "Update an existing item", { id: z.string(), value: z.number() }, async (args) => { ... });
server.tool("delete_item", "Delete an item by ID", { id: z.string() }, async (args) => { ... });`

	return Suggestion{
		ToolName:        toolName,
		Pattern:         PatternDecompose,
		Title:           fmt.Sprintf("Decompose Monolithic Tool `%s`", toolName),
		Description:     "Replace polymorphic action/mode parameter dispatcher with discrete, single-purpose MCP tools.",
		EstimatedSaving: "30% prompt schema overhead reduction & higher LLM accuracy",
		TypeScriptCode:  tsCode,
	}
}

func generateEnumResourceSuggestion(toolName string, tool mcp.Tool, hasTool bool) Suggestion {
	tsCode := fmt.Sprintf(`// Move large enum (>30 items) to an MCP Resource URI
server.resource(
  "enum-options",
  "resource://options/%s",
  async (uri) => ({
    contents: [{
      uri: uri.href,
      text: JSON.stringify(ALL_OPTIONS),
      mimeType: "application/json"
    }]
  })
);`, toolName)

	return Suggestion{
		ToolName:        toolName,
		Pattern:         PatternEnumResource,
		Title:           fmt.Sprintf("Extract Large Enum from `%s` into MCP Resource", toolName),
		Description:     "Move static 30+ item enum arrays out of the active tool schema into an on-demand MCP Resource URI.",
		EstimatedSaving: "100 - 500 tokens saved on EVERY model turn",
		TypeScriptCode:  tsCode,
	}
}

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	return strings.Join(parts, "")
}
