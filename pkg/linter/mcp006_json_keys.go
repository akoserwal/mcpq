package linter

import (
	"fmt"
	"strings"
)

type RuleMCP006JSONKeys struct{}

func (r *RuleMCP006JSONKeys) ID() string {
	return "MCP006"
}

func (r *RuleMCP006JSONKeys) Name() string {
	return "json-key-inefficiency"
}

func (r *RuleMCP006JSONKeys) Severity() Severity {
	return SeverityWarning
}

func (r *RuleMCP006JSONKeys) Description() string {
	return "JSON property names use redundant class prefixes or overly verbose identifiers (>25 chars), bloating schema and payloads."
}

func (r *RuleMCP006JSONKeys) CheckStatic(ctx *StaticContext) []Finding {
	var findings []Finding

	for _, tool := range ctx.Tools {
		var redundantKeys []string
		propNames := make([]string, 0, len(tool.InputSchema.Properties))
		for p := range tool.InputSchema.Properties {
			propNames = append(propNames, p)
		}

		// Check for common prefix redundancy (e.g. "customer_id", "customer_name", "customer_email" across >= 3 fields)
		prefixCount := make(map[string]int)
		for _, name := range propNames {
			parts := strings.Split(name, "_")
			if len(parts) > 1 {
				prefixCount[parts[0]]++
			}
			if len(name) > 28 {
				redundantKeys = append(redundantKeys, name)
			}
		}

		for prefix, count := range prefixCount {
			if count >= 3 && len(prefix) >= 4 {
				redundantKeys = append(redundantKeys, fmt.Sprintf("prefix %q used in %d properties", prefix, count))
			}
		}

		if len(redundantKeys) > 0 {
			findings = append(findings, Finding{
				RuleID:         r.ID(),
				RuleName:       r.Name(),
				Severity:       r.Severity(),
				TargetType:     "tool",
				TargetName:     tool.Name,
				Message:        fmt.Sprintf("Tool %q contains inefficient JSON key naming patterns: %s.", tool.Name, strings.Join(redundantKeys, ", ")),
				Recommendation: "Nest related fields in an object or use concise idiomatic key names without repetitive prefixes.",
				TokenImpact:    len(redundantKeys) * 15,
			})
		}
	}

	return findings
}
