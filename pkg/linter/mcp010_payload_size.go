package linter

import (
	"fmt"
)

type RuleMCP010PayloadSize struct{}

func (r *RuleMCP010PayloadSize) ID() string {
	return "MCP010"
}

func (r *RuleMCP010PayloadSize) Name() string {
	return "excessive-single-payload"
}

func (r *RuleMCP010PayloadSize) Severity() Severity {
	return SeverityError
}

func (r *RuleMCP010PayloadSize) Description() string {
	return "Individual tool response exceeds 8,000 tokens in a single call, exhausting model context and degrading reasoning."
}

func (r *RuleMCP010PayloadSize) CheckDynamic(ctx *DynamicContext) []Finding {
	var findings []Finding

	for _, trace := range ctx.Traces {
		if trace.Response == nil || len(trace.Response.Result) == 0 {
			continue
		}

		tokCount := trace.TokenCount
		if tokCount == 0 && ctx.Tokenizer != nil {
			tokCount = ctx.Tokenizer.CountTokens(string(trace.Response.Result))
		}

		if tokCount > 8000 {
			toolName := trace.ToolName
			if toolName == "" {
				toolName = trace.Method
			}

			findings = append(findings, Finding{
				RuleID:         r.ID(),
				RuleName:       r.Name(),
				Severity:       r.Severity(),
				TargetType:     "trace",
				TargetName:     toolName,
				Message:        fmt.Sprintf("Tool call to %q returned %d tokens in a single response (exceeds 8,000 token limit).", toolName, tokCount),
				Recommendation: "Paginate responses, truncate large collections, or provide summary-first endpoints with detail drill-down.",
				TokenImpact:    tokCount - 2000,
			})
		}
	}

	return findings
}
