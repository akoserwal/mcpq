package linter

import (
	"time"

	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

// Severity represents finding severity.
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
)

// Finding represents a single linter diagnostic.
type Finding struct {
	RuleID         string   `json:"ruleId"`
	RuleName       string   `json:"ruleName"`
	Severity       Severity `json:"severity"`
	TargetType     string   `json:"targetType"` // "tool", "resource", "prompt", "trace"
	TargetName     string   `json:"targetName"`
	Message        string   `json:"message"`
	Recommendation string   `json:"recommendation"`
	TokenImpact    int      `json:"tokenImpact,omitempty"`
}

// TraceEvent represents a recorded JSON-RPC interaction.
type TraceEvent struct {
	Timestamp  time.Time     `json:"timestamp"`
	Direction  string        `json:"direction"` // "request", "response"
	Method     string        `json:"method"`
	ToolName   string        `json:"toolName,omitempty"`
	Request    *mcp.Request  `json:"request,omitempty"`
	Response   *mcp.Response `json:"response,omitempty"`
	DurationMs int64         `json:"durationMs,omitempty"`
	TokenCount int           `json:"tokenCount"`
	RawBytes   int           `json:"rawBytes"`
}

// StaticContext contains context for static schema analysis.
type StaticContext struct {
	Tools     []mcp.Tool
	Resources []mcp.Resource
	Prompts   []mcp.Prompt
	Tokenizer tokenizer.Tokenizer
}

// DynamicContext contains context for dynamic trace evaluation.
type DynamicContext struct {
	Traces    []TraceEvent
	Tokenizer tokenizer.Tokenizer
}

// Rule is the common interface for lint rules.
type Rule interface {
	ID() string
	Name() string
	Severity() Severity
	Description() string
}

// StaticRule is implemented by rules that analyze static MCP definitions.
type StaticRule interface {
	Rule
	CheckStatic(ctx *StaticContext) []Finding
}

// DynamicRule is implemented by rules that evaluate dynamic trace logs.
type DynamicRule interface {
	Rule
	CheckDynamic(ctx *DynamicContext) []Finding
}
