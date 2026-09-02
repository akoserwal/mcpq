package linter

import (
	"fmt"
	"strings"
)

type RuleMCP002Monolithic struct{}

func (r *RuleMCP002Monolithic) ID() string {
	return "MCP002"
}

func (r *RuleMCP002Monolithic) Name() string {
	return "monolithic-tool"
}

func (r *RuleMCP002Monolithic) Severity() Severity {
	return SeverityError
}

func (r *RuleMCP002Monolithic) Description() string {
	return "Tool has > 8 parameters or acts as an overloaded multi-action dispatcher, hurting model reasoning accuracy."
}

func (r *RuleMCP002Monolithic) CheckStatic(ctx *StaticContext) []Finding {
	var findings []Finding

	for _, tool := range ctx.Tools {
		paramCount := len(tool.InputSchema.Properties)
		
		// Check for overloaded dispatcher parameters
		hasDispatcher := false
		for pName, prop := range tool.InputSchema.Properties {
			lower := strings.ToLower(pName)
			if lower == "action" || lower == "operation" || lower == "command" || lower == "mode" || lower == "method" {
				if len(prop.Enum) >= 3 {
					hasDispatcher = true
					break
				}
			}
		}

		if paramCount > 8 {
			findings = append(findings, Finding{
				RuleID:         r.ID(),
				RuleName:       r.Name(),
				Severity:       r.Severity(),
				TargetType:     "tool",
				TargetName:     tool.Name,
				Message:        fmt.Sprintf("Tool %q defines %d parameters (threshold: <= 8). High parameter complexity increases agent confusion and prompt size.", tool.Name, paramCount),
				Recommendation: "Decompose monolithic tool into discrete, focused tools with fewer arguments.",
				TokenImpact:    paramCount * 15,
			})
		} else if hasDispatcher && paramCount >= 5 {
			findings = append(findings, Finding{
				RuleID:         r.ID(),
				RuleName:       r.Name(),
				Severity:       r.Severity(),
				TargetType:     "tool",
				TargetName:     tool.Name,
				Message:        fmt.Sprintf("Tool %q acts as a multi-action dispatcher with %d parameters.", tool.Name, paramCount),
				Recommendation: "Split polymorphic action dispatchers into dedicated individual tools (e.g. 'create_user', 'update_user' instead of 'manage_user(action=create|update)').",
				TokenImpact:    paramCount * 10,
			})
		}
	}

	return findings
}
