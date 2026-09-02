package linter

import (
	"fmt"
)

type RuleMCP001Description struct{}

func (r *RuleMCP001Description) ID() string {
	return "MCP001"
}

func (r *RuleMCP001Description) Name() string {
	return "huge-tool-description"
}

func (r *RuleMCP001Description) Severity() Severity {
	return SeverityError
}

func (r *RuleMCP001Description) Description() string {
	return "Tool description exceeds 1,000 characters or 250 tokens, bloating the initial context window."
}

func (r *RuleMCP001Description) CheckStatic(ctx *StaticContext) []Finding {
	var findings []Finding

	for _, tool := range ctx.Tools {
		desc := tool.Description
		charLen := len(desc)
		tokenLen := 0
		if ctx.Tokenizer != nil {
			tokenLen = ctx.Tokenizer.CountTokens(desc)
		} else {
			tokenLen = charLen / 4
		}

		if charLen > 1000 || tokenLen > 250 {
			findings = append(findings, Finding{
				RuleID:         r.ID(),
				RuleName:       r.Name(),
				Severity:       r.Severity(),
				TargetType:     "tool",
				TargetName:     tool.Name,
				Message:        fmt.Sprintf("Tool %q description is excessively large (%d chars, ~%d tokens).", tool.Name, charLen, tokenLen),
				Recommendation: "Condense description to 1-3 concise sentences. Move detailed tutorials or documentation into MCP Prompts or Resources.",
				TokenImpact:    tokenLen - 50,
			})
		}
	}

	return findings
}
