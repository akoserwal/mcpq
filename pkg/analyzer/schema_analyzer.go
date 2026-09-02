package analyzer

import (
	"encoding/json"
	"math"

	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

// ToolSchemaStats holds metrics for a single tool.
type ToolSchemaStats struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	DescriptionLen int    `json:"descriptionLength"`
	TokenCount     int    `json:"tokenCount"`
	ParamCount     int    `json:"paramCount"`
	RequiredCount  int    `json:"requiredCount"`
	EnumCount      int    `json:"enumCount"`
}

// SchemaAnalysis holds comprehensive static schema metrics.
type SchemaAnalysis struct {
	ToolCount        int               `json:"toolCount"`
	ResourceCount    int               `json:"resourceCount"`
	PromptCount      int               `json:"promptCount"`
	TotalTokens      int               `json:"totalTokens"`
	AvgTokensPerTool float64           `json:"avgTokensPerTool"`
	MinTokensPerTool int               `json:"minTokensPerTool"`
	MaxTokensPerTool int               `json:"maxTokensPerTool"`
	TotalParams      int               `json:"totalParams"`
	AvgParamsPerTool float64           `json:"avgParamsPerTool"`
	ToolStats        []ToolSchemaStats `json:"toolStats"`
}

// AnalyzeSchema performs static analysis on MCP tools, resources, and prompts.
func AnalyzeSchema(tools []mcp.Tool, resources []mcp.Resource, prompts []mcp.Prompt, tok tokenizer.Tokenizer) *SchemaAnalysis {
	if tok == nil {
		tok = tokenizer.DefaultTokenizer()
	}

	analysis := &SchemaAnalysis{
		ToolCount:        len(tools),
		ResourceCount:    len(resources),
		PromptCount:      len(prompts),
		MinTokensPerTool: math.MaxInt32,
		MaxTokensPerTool: 0,
		ToolStats:        make([]ToolSchemaStats, 0, len(tools)),
	}

	if len(tools) == 0 {
		analysis.MinTokensPerTool = 0
		return analysis
	}

	totalTokens := 0
	totalParams := 0

	for _, tool := range tools {
		toolJSON, _ := json.Marshal(tool)
		toolTokens := tok.CountTokens(string(toolJSON))
		totalTokens += toolTokens

		paramCount := len(tool.InputSchema.Properties)
		totalParams += paramCount

		enumCount := 0
		for _, prop := range tool.InputSchema.Properties {
			if len(prop.Enum) > 0 {
				enumCount += len(prop.Enum)
			}
			if prop.Items != nil && len(prop.Items.Enum) > 0 {
				enumCount += len(prop.Items.Enum)
			}
		}

		if toolTokens < analysis.MinTokensPerTool {
			analysis.MinTokensPerTool = toolTokens
		}
		if toolTokens > analysis.MaxTokensPerTool {
			analysis.MaxTokensPerTool = toolTokens
		}

		analysis.ToolStats = append(analysis.ToolStats, ToolSchemaStats{
			Name:           tool.Name,
			Description:    tool.Description,
			DescriptionLen: len(tool.Description),
			TokenCount:     toolTokens,
			ParamCount:     paramCount,
			RequiredCount:  len(tool.InputSchema.Required),
			EnumCount:      enumCount,
		})
	}

	analysis.TotalTokens = totalTokens
	analysis.TotalParams = totalParams
	analysis.AvgTokensPerTool = float64(totalTokens) / float64(len(tools))
	analysis.AvgParamsPerTool = float64(totalParams) / float64(len(tools))

	return analysis
}
