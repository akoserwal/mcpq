package linter

import (
	"fmt"
	"strings"
)

type RuleMCP009Docstring struct{}

func (r *RuleMCP009Docstring) ID() string {
	return "MCP009"
}

func (r *RuleMCP009Docstring) Name() string {
	return "missing-property-descriptions"
}

func (r *RuleMCP009Docstring) Severity() Severity {
	return SeverityInfo
}

func (r *RuleMCP009Docstring) Description() string {
	return "Tool parameters lack descriptive documentation, increasing LLM uncertainty and hallucinated argument values."
}

func (r *RuleMCP009Docstring) CheckStatic(ctx *StaticContext) []Finding {
	var findings []Finding

	for _, tool := range ctx.Tools {
		var missingProps []string
		for propName, prop := range tool.InputSchema.Properties {
			if strings.TrimSpace(prop.Description) == "" {
				missingProps = append(missingProps, propName)
			}
		}

		if len(missingProps) > 0 {
			findings = append(findings, Finding{
				RuleID:         r.ID(),
				RuleName:       r.Name(),
				Severity:       r.Severity(),
				TargetType:     "tool",
				TargetName:     tool.Name,
				Message:        fmt.Sprintf("Tool %q has %d parameter(s) missing descriptions: %s.", tool.Name, len(missingProps), strings.Join(missingProps, ", ")),
				Recommendation: "Provide concise descriptions and types for all parameters to ensure reliable agent tool usage.",
				TokenImpact:    0,
			})
		}
	}

	return findings
}
