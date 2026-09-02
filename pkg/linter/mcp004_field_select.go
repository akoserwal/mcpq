package linter

import (
	"fmt"
	"strings"
)

type RuleMCP004FieldSelect struct{}

func (r *RuleMCP004FieldSelect) ID() string {
	return "MCP004"
}

func (r *RuleMCP004FieldSelect) Name() string {
	return "no-field-selection"
}

func (r *RuleMCP004FieldSelect) Severity() Severity {
	return SeverityWarning
}

func (r *RuleMCP004FieldSelect) Description() string {
	return "Data retrieval tool lacks field filtering parameters (fields/select/include), returning full bloated payloads when only a few fields are needed."
}

func (r *RuleMCP004FieldSelect) CheckStatic(ctx *StaticContext) []Finding {
	var findings []Finding

	projectionKeywords := map[string]bool{
		"fields":       true,
		"select":       true,
		"include":      true,
		"columns":      true,
		"properties":   true,
		"view":         true,
		"detail_level": true,
		"summary":      true,
	}

	for _, tool := range ctx.Tools {
		lowerName := strings.ToLower(tool.Name)
		isFetch := strings.HasPrefix(lowerName, "get_") ||
			strings.HasPrefix(lowerName, "read_") ||
			strings.HasPrefix(lowerName, "fetch_") ||
			strings.HasPrefix(lowerName, "view_") ||
			strings.HasPrefix(lowerName, "inspect_")

		// Skip simple scalar lookups like get_status or get_version
		if strings.HasSuffix(lowerName, "_status") ||
			strings.HasSuffix(lowerName, "_version") ||
			strings.HasSuffix(lowerName, "_health") ||
			strings.HasSuffix(lowerName, "_count") {
			continue
		}

		if !isFetch {
			continue
		}

		hasProjection := false
		for propName := range tool.InputSchema.Properties {
			if projectionKeywords[strings.ToLower(propName)] {
				hasProjection = true
				break
			}
		}

		if !hasProjection {
			findings = append(findings, Finding{
				RuleID:         r.ID(),
				RuleName:       r.Name(),
				Severity:       r.Severity(),
				TargetType:     "tool",
				TargetName:     tool.Name,
				Message:        fmt.Sprintf("Entity retrieval tool %q lacks field filtering/projection arguments (e.g. 'fields', 'select').", tool.Name),
				Recommendation: "Add an optional 'fields?: string[]' or 'view?: summary|detail' parameter so LLM agents can request only relevant attributes.",
				TokenImpact:    500,
			})
		}
	}

	return findings
}
