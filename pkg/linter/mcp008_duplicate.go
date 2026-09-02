package linter

import (
	"fmt"
	"strings"

	"github.com/akoserwal/mcpq/pkg/mcp"
)

type RuleMCP008Duplicate struct{}

func (r *RuleMCP008Duplicate) ID() string {
	return "MCP008"
}

func (r *RuleMCP008Duplicate) Name() string {
	return "redundant-tool-schema"
}

func (r *RuleMCP008Duplicate) Severity() Severity {
	return SeverityWarning
}

func (r *RuleMCP008Duplicate) Description() string {
	return "Multiple tools share >= 80% parameter/description overlap without clear functional separation, confusing LLM tool routing."
}

func (r *RuleMCP008Duplicate) CheckStatic(ctx *StaticContext) []Finding {
	var findings []Finding
	tools := ctx.Tools
	n := len(tools)

	if n < 2 {
		return findings
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			t1 := tools[i]
			t2 := tools[j]

			similarity := computeToolSimilarity(t1, t2)
			if similarity >= 0.80 {
				findings = append(findings, Finding{
					RuleID:         r.ID(),
					RuleName:       r.Name(),
					Severity:       r.Severity(),
					TargetType:     "tool",
					TargetName:     fmt.Sprintf("%s / %s", t1.Name, t2.Name),
					Message:        fmt.Sprintf("Tools %q and %q have %.0f%% schema/description similarity.", t1.Name, t2.Name, similarity*100),
					Recommendation: "Consolidate duplicate tools into a unified operation or clarify distinctive tool naming and documentation.",
					TokenImpact:    100,
				})
			}
		}
	}

	return findings
}

func computeToolSimilarity(t1, t2 mcp.Tool) float64 {
	p1 := t1.InputSchema.Properties
	p2 := t2.InputSchema.Properties

	if len(p1) == 0 && len(p2) == 0 {
		return 0.0
	}

	intersection := 0
	allKeys := make(map[string]bool)

	for k := range p1 {
		allKeys[k] = true
	}
	for k := range p2 {
		if allKeys[k] {
			intersection++
		}
		allKeys[k] = true
	}

	union := len(allKeys)
	if union == 0 {
		return 0.0
	}

	propJaccard := float64(intersection) / float64(union)

	// Check description word overlap
	descOverlap := wordJaccard(t1.Description, t2.Description)

	return (propJaccard * 0.7) + (descOverlap * 0.3)
}

func wordJaccard(s1, s2 string) float64 {
	w1 := strings.Fields(strings.ToLower(s1))
	w2 := strings.Fields(strings.ToLower(s2))

	if len(w1) == 0 || len(w2) == 0 {
		return 0.0
	}

	m1 := make(map[string]bool)
	for _, w := range w1 {
		m1[w] = true
	}

	inter := 0
	all := make(map[string]bool)
	for _, w := range w1 {
		all[w] = true
	}
	for _, w := range w2 {
		if m1[w] {
			inter++
		}
		all[w] = true
	}

	if len(all) == 0 {
		return 0.0
	}
	return float64(inter) / float64(len(all))
}
