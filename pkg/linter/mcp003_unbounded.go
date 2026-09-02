package linter

import (
	"fmt"
	"strings"
)

type RuleMCP003Unbounded struct{}

func (r *RuleMCP003Unbounded) ID() string {
	return "MCP003"
}

func (r *RuleMCP003Unbounded) Name() string {
	return "unbounded-response"
}

func (r *RuleMCP003Unbounded) Severity() Severity {
	return SeverityError
}

func (r *RuleMCP003Unbounded) Description() string {
	return "Collection/query tool lacks pagination parameters (limit/cursor/page/offset), risking unbounded token payloads."
}

func (r *RuleMCP003Unbounded) CheckStatic(ctx *StaticContext) []Finding {
	var findings []Finding

	paginationKeywords := map[string]bool{
		"limit":       true,
		"cursor":      true,
		"page":        true,
		"offset":      true,
		"page_size":   true,
		"pageSize":    true,
		"max_results": true,
		"maxResults":  true,
		"count":       true,
		"top":         true,
	}

	for _, tool := range ctx.Tools {
		lowerName := strings.ToLower(tool.Name)
		isCollection := strings.HasPrefix(lowerName, "list") ||
			strings.HasPrefix(lowerName, "search") ||
			strings.HasPrefix(lowerName, "find") ||
			strings.HasPrefix(lowerName, "query") ||
			strings.HasPrefix(lowerName, "browse") ||
			strings.HasSuffix(lowerName, "_all") ||
			strings.Contains(lowerName, "get_all")

		if !isCollection {
			continue
		}

		hasPagination := false
		for propName := range tool.InputSchema.Properties {
			if paginationKeywords[strings.ToLower(propName)] {
				hasPagination = true
				break
			}
		}

		if !hasPagination {
			findings = append(findings, Finding{
				RuleID:         r.ID(),
				RuleName:       r.Name(),
				Severity:       r.Severity(),
				TargetType:     "tool",
				TargetName:     tool.Name,
				Message:        fmt.Sprintf("Collection tool %q does not accept any pagination controls (e.g. limit, cursor, page).", tool.Name),
				Recommendation: "Add 'limit' (with default like 20) and 'cursor'/'page' parameters to enable progressive context retrieval.",
				TokenImpact:    2000,
			})
		}
	}

	return findings
}
