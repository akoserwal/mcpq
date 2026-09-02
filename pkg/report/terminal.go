package report

import (
	"fmt"
	"io"

	"github.com/akoserwal/mcpq/pkg/analyzer"
	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/scorer"
)

// ANSI color codes
const (
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[90m"
	BgRed        = "\033[41m\033[37m"
	BgGreen      = "\033[42m\033[30m"
	BgYellow     = "\033[43m\033[30m"
)

// RenderTerminal writes a formatted CLI report to w.
func RenderTerminal(w io.Writer, schema *analyzer.SchemaAnalysis, traces *analyzer.TraceAnalysis, score *scorer.QualityScore) {
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "%s%s=================================================================%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Fprintf(w, "%s%s                 MCPQ - MCP QUALITY & CONTEXT AUDIT             %s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Fprintf(w, "%s%s=================================================================%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Fprintln(w, "")

	// 1. Overall Score & Badge
	badgeColor := ColorGreen
	switch score.Grade {
	case scorer.GradeAPlus, scorer.GradeA:
		badgeColor = ColorGreen
	case scorer.GradeB:
		badgeColor = ColorCyan
	case scorer.GradeC:
		badgeColor = ColorYellow
	default:
		badgeColor = ColorRed
	}

	fmt.Fprintf(w, "  %sOverall Score:%s %s%s%d / 100%s  [%sGrade %s%s]\n",
		ColorBold, ColorReset, badgeColor, ColorBold, score.Score, ColorReset, badgeColor, score.Grade, ColorReset)
	fmt.Fprintf(w, "  %sSummary:%s       %s\n", ColorGray, ColorReset, score.Summary)
	fmt.Fprintln(w, "")

	// 2. Sub-Score Breakdown Table
	fmt.Fprintf(w, "  %s--- Multi-Dimensional Quality Index ---%s\n", ColorBold, ColorReset)
	fmt.Fprintf(w, "  %-26s : %s%4.1f / 20.0%s\n", "Schema Efficiency", ColorCyan, score.Breakdown.SchemaEfficiency, ColorReset)
	fmt.Fprintf(w, "  %-26s : %s%4.1f / 20.0%s\n", "Tool Design & Modularity", ColorCyan, score.Breakdown.ToolDesign, ColorReset)
	fmt.Fprintf(w, "  %-26s : %s%4.1f / 30.0%s\n", "Response Efficiency", ColorCyan, score.Breakdown.ResponseEfficiency, ColorReset)
	fmt.Fprintf(w, "  %-26s : %s%4.1f / 15.0%s\n", "Progressive Disclosure", ColorCyan, score.Breakdown.ProgressiveDisclosure, ColorReset)
	fmt.Fprintf(w, "  %-26s : %s%4.1f / 10.0%s\n", "Discoverability", ColorCyan, score.Breakdown.Discoverability, ColorReset)
	if score.Breakdown.Penalties > 0 {
		fmt.Fprintf(w, "  %-26s : %s-%4.1f pts%s\n", "Rule Penalties", ColorRed, score.Breakdown.Penalties, ColorReset)
	}
	fmt.Fprintln(w, "")

	// 3. Schema Metrics
	if schema != nil && schema.ToolCount > 0 {
		fmt.Fprintf(w, "  %s--- Static Schema Analysis ---%s\n", ColorBold, ColorReset)
		fmt.Fprintf(w, "  Tools: %-4d | Resources: %-3d | Prompts: %-3d\n", schema.ToolCount, schema.ResourceCount, schema.PromptCount)
		fmt.Fprintf(w, "  Total Schema Tokens: %-6d (Avg: %.1f tokens/tool, Min: %d, Max: %d)\n",
			schema.TotalTokens, schema.AvgTokensPerTool, schema.MinTokensPerTool, schema.MaxTokensPerTool)
		fmt.Fprintf(w, "  Parameters:          %-6d (Avg: %.1f params/tool)\n", schema.TotalParams, schema.AvgParamsPerTool)
		fmt.Fprintln(w, "")

		fmt.Fprintf(w, "  %-24s %-10s %-8s %-12s\n", "TOOL NAME", "TOKENS", "PARAMS", "DESCRIPTION")
		fmt.Fprintf(w, "  %-24s %-10s %-8s %-12s\n", "------------------------", "----------", "--------", "------------")
		for _, t := range schema.ToolStats {
			descPreview := t.Description
			if len(descPreview) > 35 {
				descPreview = descPreview[:32] + "..."
			}
			if descPreview == "" {
				descPreview = fmt.Sprintf("%s(missing)%s", ColorRed, ColorReset)
			}
			fmt.Fprintf(w, "  %-24s %-10d %-8d %s\n", truncate(t.Name, 24), t.TokenCount, t.ParamCount, descPreview)
		}
		fmt.Fprintln(w, "")
	}

	// 4. Trace Metrics (if present)
	if traces != nil && traces.TotalToolCalls > 0 {
		fmt.Fprintf(w, "  %s--- Dynamic Runtime Trace Analysis ---%s\n", ColorBold, ColorReset)
		fmt.Fprintf(w, "  Recorded Calls: %-4d | Total Tokens: %-7d (Req: %d, Resp: %d)\n",
			traces.TotalToolCalls, traces.TotalTokens, traces.TotalRequestTokens, traces.TotalRespTokens)
		fmt.Fprintf(w, "  Token Distribution : P50: %-5d | P90: %-5d | P95: %-5d | Max: %-5d\n",
			traces.P50Tokens, traces.P90Tokens, traces.P95Tokens, traces.MaxTokens)
		fmt.Fprintf(w, "  Metadata Waste Est : %s%.1f%%%s of response payloads\n",
			ColorYellow, traces.MetadataWasteRatio*100, ColorReset)
		if traces.AvgDurationMs > 0 {
			fmt.Fprintf(w, "  Latency            : Avg: %.1fms | P95: %dms\n", traces.AvgDurationMs, traces.P95DurationMs)
		}
		fmt.Fprintln(w, "")
	}

	// 5. Findings & Anti-Patterns
	fmt.Fprintf(w, "  %s--- Lint Findings & Anti-Patterns (%d) ---%s\n", ColorBold, len(score.Findings), ColorReset)
	if len(score.Findings) == 0 {
		fmt.Fprintf(w, "  %s✓ No anti-patterns detected! Clean progressive disclosure.%s\n", ColorGreen, ColorReset)
	} else {
		for i, f := range score.Findings {
			sevBadge := fmt.Sprintf("%s[ERROR]%s", ColorRed, ColorReset)
			switch f.Severity {
			case linter.SeverityWarning:
				sevBadge = fmt.Sprintf("%s[WARN]%s ", ColorYellow, ColorReset)
			case linter.SeverityInfo:
				sevBadge = fmt.Sprintf("%s[INFO]%s ", ColorBlue, ColorReset)
			}

			fmt.Fprintf(w, "  %d. %s %s%s%s (%s: %s%s%s)\n",
				i+1, sevBadge, ColorBold, f.RuleID, ColorReset, f.TargetType, ColorCyan, f.TargetName, ColorReset)
			fmt.Fprintf(w, "     %s%s%s\n", ColorGray, f.Message, ColorReset)
			fmt.Fprintf(w, "     %sRecommendation:%s %s\n", ColorGreen, ColorReset, f.Recommendation)
			if f.TokenImpact > 0 {
				fmt.Fprintf(w, "     %sEstimated Token Impact:%s ~%d tokens\n", ColorYellow, ColorReset, f.TokenImpact)
			}
			fmt.Fprintln(w, "")
		}
	}

	fmt.Fprintf(w, "%s=================================================================%s\n", ColorCyan, ColorReset)
	fmt.Fprintln(w, "")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
