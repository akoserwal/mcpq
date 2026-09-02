package linter

import (
	"fmt"
)

type RuleMCP007EnumBloat struct{}

func (r *RuleMCP007EnumBloat) ID() string {
	return "MCP007"
}

func (r *RuleMCP007EnumBloat) Name() string {
	return "enum-explosion"
}

func (r *RuleMCP007EnumBloat) Severity() Severity {
	return SeverityWarning
}

func (r *RuleMCP007EnumBloat) Description() string {
	return "Tool schema contains large enum lists (>30 entries) directly inline, bloating prompt size on every interaction."
}

func (r *RuleMCP007EnumBloat) CheckStatic(ctx *StaticContext) []Finding {
	var findings []Finding

	for _, tool := range ctx.Tools {
		for propName, prop := range tool.InputSchema.Properties {
			enumLen := len(prop.Enum)
			if prop.Items != nil && len(prop.Items.Enum) > enumLen {
				enumLen = len(prop.Items.Enum)
			}

			if enumLen > 30 {
				tokenCost := enumLen * 3
				findings = append(findings, Finding{
					RuleID:         r.ID(),
					RuleName:       r.Name(),
					Severity:       r.Severity(),
					TargetType:     "tool",
					TargetName:     tool.Name,
					Message:        fmt.Sprintf("Parameter %q in tool %q defines an enum with %d values.", propName, tool.Name, enumLen),
					Recommendation: "Extract large static enum lookups (>30 items) to an MCP resource URI (e.g. 'resource://options/categories') or a searchable lookup tool.",
					TokenImpact:    tokenCost,
				})
			}
		}
	}

	return findings
}
