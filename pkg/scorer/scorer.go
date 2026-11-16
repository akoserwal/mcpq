package scorer

import (
	"fmt"
	"math"

	"github.com/akoserwal/mcpq/pkg/analyzer"
	"github.com/akoserwal/mcpq/pkg/linter"
)

// CalculateScore computes the MCP Quality Score from static and dynamic analysis.
func CalculateScore(schema *analyzer.SchemaAnalysis, traces *analyzer.TraceAnalysis, findings []linter.Finding) *QualityScore {
	breakdown := ScoreBreakdown{}

	// 1. Schema Efficiency (Max 20 pts)
	if schema != nil && schema.ToolCount > 0 {
		// Avg tokens per tool (max 10 pts)
		switch {
		case schema.AvgTokensPerTool <= 120:
			breakdown.SchemaEfficiency += 10
		case schema.AvgTokensPerTool <= 200:
			breakdown.SchemaEfficiency += 8
		case schema.AvgTokensPerTool <= 350:
			breakdown.SchemaEfficiency += 5
		case schema.AvgTokensPerTool <= 500:
			breakdown.SchemaEfficiency += 2
		default:
			breakdown.SchemaEfficiency += 0
		}

		// Total schema tokens across all tools (max 10 pts)
		switch {
		case schema.TotalTokens <= 1500:
			breakdown.SchemaEfficiency += 10
		case schema.TotalTokens <= 3000:
			breakdown.SchemaEfficiency += 8
		case schema.TotalTokens <= 6000:
			breakdown.SchemaEfficiency += 5
		case schema.TotalTokens <= 10000:
			breakdown.SchemaEfficiency += 2
		default:
			breakdown.SchemaEfficiency += 0
		}
	} else {
		breakdown.SchemaEfficiency = 20
	}

	// 2. Tool Design & Modularity (Max 20 pts)
	if schema != nil && schema.ToolCount > 0 {
		// Parameter count per tool (max 10 pts)
		switch {
		case schema.AvgParamsPerTool <= 3.5:
			breakdown.ToolDesign += 10
		case schema.AvgParamsPerTool <= 5.0:
			breakdown.ToolDesign += 8
		case schema.AvgParamsPerTool <= 7.0:
			breakdown.ToolDesign += 5
		case schema.AvgParamsPerTool <= 9.0:
			breakdown.ToolDesign += 2
		default:
			breakdown.ToolDesign += 0
		}

		// Check for clean property types & absence of monolithic dispatcher
		hasMonolithic := false
		for _, f := range findings {
			if f.RuleID == "MCP002" || f.RuleID == "MCP007" {
				hasMonolithic = true
				break
			}
		}
		if !hasMonolithic {
			breakdown.ToolDesign += 10
		} else {
			breakdown.ToolDesign += 4
		}
	} else {
		breakdown.ToolDesign = 20
	}

	// 3. Response Efficiency (Max 30 pts)
	if traces != nil && traces.TotalToolCalls > 0 {
		// P50 response tokens (max 12 pts)
		switch {
		case traces.P50Tokens <= 800:
			breakdown.ResponseEfficiency += 12
		case traces.P50Tokens <= 1500:
			breakdown.ResponseEfficiency += 9
		case traces.P50Tokens <= 3000:
			breakdown.ResponseEfficiency += 5
		case traces.P50Tokens <= 6000:
			breakdown.ResponseEfficiency += 2
		default:
			breakdown.ResponseEfficiency += 0
		}

		// Envelope waste ratio (max 10 pts)
		switch {
		case traces.MetadataWasteRatio <= 0.05:
			breakdown.ResponseEfficiency += 10
		case traces.MetadataWasteRatio <= 0.12:
			breakdown.ResponseEfficiency += 7
		case traces.MetadataWasteRatio <= 0.20:
			breakdown.ResponseEfficiency += 4
		default:
			breakdown.ResponseEfficiency += 0
		}

		// P95 response tokens (max 8 pts)
		switch {
		case traces.P95Tokens <= 2500:
			breakdown.ResponseEfficiency += 8
		case traces.P95Tokens <= 5000:
			breakdown.ResponseEfficiency += 5
		case traces.P95Tokens <= 8000:
			breakdown.ResponseEfficiency += 2
		default:
			breakdown.ResponseEfficiency += 0
		}
	}

	// 4. Progressive Disclosure (Max 15 pts)
	hasPaginationRule := false
	hasFieldSelectRule := false
	for _, f := range findings {
		if f.RuleID == "MCP003" {
			hasPaginationRule = true
		}
		if f.RuleID == "MCP004" {
			hasFieldSelectRule = true
		}
	}

	if !hasPaginationRule {
		breakdown.ProgressiveDisclosure += 8
	} else {
		breakdown.ProgressiveDisclosure += 2
	}

	if !hasFieldSelectRule {
		breakdown.ProgressiveDisclosure += 7
	} else {
		breakdown.ProgressiveDisclosure += 2
	}

	// 5. Discoverability (Max 10 pts)
	if schema != nil && schema.ToolCount > 0 {
		missingDocs := 0
		hugeDocs := 0
		for _, tool := range schema.ToolStats {
			if tool.DescriptionLen < 10 {
				missingDocs++
			}
			if tool.DescriptionLen > 1000 {
				hugeDocs++
			}
		}
		hasDocRule := false
		for _, f := range findings {
			if f.RuleID == "MCP009" {
				hasDocRule = true
				break
			}
		}

		if missingDocs == 0 && !hasDocRule {
			breakdown.Discoverability += 5
		} else {
			breakdown.Discoverability += 2
		}

		if hugeDocs == 0 {
			breakdown.Discoverability += 5
		} else {
			breakdown.Discoverability += 1
		}
	} else {
		breakdown.Discoverability = 10
	}

	// 6. Penalties
	for _, f := range findings {
		switch f.Severity {
		case linter.SeverityError:
			breakdown.Penalties += 10
		case linter.SeverityWarning:
			breakdown.Penalties += 4
		case linter.SeverityInfo:
			breakdown.Penalties += 1
		}
	}

	// Normalize only the dimensions supported by the supplied evidence.
	hasSchema := schema != nil && schema.ToolCount > 0
	hasRuntime := traces != nil && traces.TotalToolCalls > 0
	scope := "none"
	measuredPoints := 0.0
	samples := 0
	if hasSchema {
		measuredPoints += 65
		scope = "static"
	} else {
		breakdown.SchemaEfficiency = 0
		breakdown.ToolDesign = 0
		breakdown.ProgressiveDisclosure = 0
		breakdown.Discoverability = 0
	}
	if hasRuntime {
		measuredPoints += 30
		samples = traces.TotalToolCalls
		if hasSchema {
			scope = "combined"
		} else {
			scope = "runtime"
		}
	}
	rawTotal := breakdown.SchemaEfficiency +
		breakdown.ToolDesign +
		breakdown.ResponseEfficiency +
		breakdown.ProgressiveDisclosure +
		breakdown.Discoverability -
		breakdown.Penalties

	finalScore := 0
	if measuredPoints > 0 {
		finalScore = int(math.Round(100 * rawTotal / measuredPoints))
	}
	if finalScore > 100 {
		finalScore = 100
	}
	if finalScore < 0 {
		finalScore = 0
	}

	grade := ComputeGrade(finalScore)

	summary := fmt.Sprintf("%s assessment; normalized over %.0f measured points. Unmeasured dimensions are excluded (shown as 0).", scope, measuredPoints)
	if !hasRuntime {
		summary += " Runtime efficiency has not been measured."
	}
	if !hasSchema {
		summary += " Schema quality has not been measured."
	}
	if samples == 1 {
		summary += " One response sample only; percentiles and latency are not representative."
	} else if samples > 1 {
		summary += fmt.Sprintf(" Based on %d response samples; results apply only to the recorded workload.", samples)
	}
	return &QualityScore{
		Scope:          scope,
		MeasuredPoints: measuredPoints,
		RuntimeSamples: samples,
		Score:          finalScore,
		Grade:          grade,
		Summary:        summary,
		Breakdown:      breakdown,
		Findings:       findings,
	}
}
