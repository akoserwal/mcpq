package report

import (
	"fmt"
	"io"

	"github.com/akoserwal/mcpq/pkg/analyzer"
	"github.com/akoserwal/mcpq/pkg/scorer"
)

// RenderMarkdown writes a GitHub-flavored Markdown report.
func RenderMarkdown(w io.Writer, schema *analyzer.SchemaAnalysis, traces *analyzer.TraceAnalysis, score *scorer.QualityScore) {
	fmt.Fprintln(w, "# 📊 MCPQ Quality & Context Efficiency Audit")
	fmt.Fprintln(w, "")

	badge := "🟢"
	switch score.Grade {
	case scorer.GradeAPlus, scorer.GradeA:
		badge = "🟢"
	case scorer.GradeB:
		badge = "🔵"
	case scorer.GradeC:
		badge = "🟡"
	default:
		badge = "🔴"
	}

	fmt.Fprintf(w, "### Overall Quality Score: **%d / 100** (%s Grade `%s`)\n\n", score.Score, badge, score.Grade)
	fmt.Fprintf(w, "> **Summary**: %s\n\n", score.Summary)

	fmt.Fprintln(w, "## 📈 Sub-Score Breakdown")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "| Dimension | Score | Max Points |")
	fmt.Fprintln(w, "| :--- | :--- | :--- |")
	fmt.Fprintf(w, "| **Schema Efficiency** | `%.1f` | 20.0 |\n", score.Breakdown.SchemaEfficiency)
	fmt.Fprintf(w, "| **Tool Design & Modularity** | `%.1f` | 20.0 |\n", score.Breakdown.ToolDesign)
	fmt.Fprintf(w, "| **Response Efficiency** | `%.1f` | 30.0 |\n", score.Breakdown.ResponseEfficiency)
	fmt.Fprintf(w, "| **Progressive Disclosure** | `%.1f` | 15.0 |\n", score.Breakdown.ProgressiveDisclosure)
	fmt.Fprintf(w, "| **Discoverability** | `%.1f` | 10.0 |\n", score.Breakdown.Discoverability)
	if score.Breakdown.Penalties > 0 {
		fmt.Fprintf(w, "| **Rule Penalties** | `-%.1f` | - |\n", score.Breakdown.Penalties)
	}
	fmt.Fprintln(w, "")

	if schema != nil && schema.ToolCount > 0 {
		fmt.Fprintln(w, "## 🛠 Static Tool Schema Summary")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "- **Tools**: %d\n", schema.ToolCount)
		fmt.Fprintf(w, "- **Total Schema Tokens**: %d (Avg: `%.1f` tokens/tool, Max: `%d` tokens)\n",
			schema.TotalTokens, schema.AvgTokensPerTool, schema.MaxTokensPerTool)
		fmt.Fprintf(w, "- **Total Parameters**: %d (Avg: `%.1f` params/tool)\n\n", schema.TotalParams, schema.AvgParamsPerTool)

		fmt.Fprintln(w, "| Tool Name | Tokens | Parameters | Description |")
		fmt.Fprintln(w, "| :--- | :--- | :--- | :--- |")
		for _, t := range schema.ToolStats {
			desc := t.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			if desc == "" {
				desc = "*missing*"
			}
			fmt.Fprintf(w, "| `%s` | %d | %d | %s |\n", t.Name, t.TokenCount, t.ParamCount, desc)
		}
		fmt.Fprintln(w, "")
	}

	if traces != nil && traces.TotalToolCalls > 0 {
		fmt.Fprintln(w, "## ⚡ Runtime Trace Analysis")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "- **Calls Recorded**: %d\n", traces.TotalToolCalls)
		fmt.Fprintf(w, "- **Total Tokens**: %d (Prompt: `%d`, Completion: `%d`)\n", traces.TotalTokens, traces.TotalRequestTokens, traces.TotalRespTokens)
		fmt.Fprintf(w, "- **Token Distribution**: P50: `%d` | P90: `%d` | P95: `%d` | Max: `%d`\n",
			traces.P50Tokens, traces.P90Tokens, traces.P95Tokens, traces.MaxTokens)
		fmt.Fprintf(w, "- **Metadata Overhead Waste**: `%.1f%%`\n\n", traces.MetadataWasteRatio*100)
	}

	fmt.Fprintf(w, "## 🚨 Diagnostics & Lint Findings (%d)\n\n", len(score.Findings))
	if len(score.Findings) == 0 {
		fmt.Fprintln(w, "✨ **No anti-patterns detected.** Tool schemas adhere to progressive disclosure best practices.")
	} else {
		for i, f := range score.Findings {
			sevEmoji := "🔴"
			if f.Severity == "WARNING" {
				sevEmoji = "🟠"
			} else if f.Severity == "INFO" {
				sevEmoji = "🟡"
			}
			fmt.Fprintf(w, "### %d. %s `%s`: %s (%s `%s`)\n\n", i+1, sevEmoji, f.RuleID, f.RuleName, f.TargetType, f.TargetName)
			fmt.Fprintf(w, "- **Message**: %s\n", f.Message)
			fmt.Fprintf(w, "- **Recommendation**: %s\n", f.Recommendation)
			if f.TokenImpact > 0 {
				fmt.Fprintf(w, "- **Estimated Token Waste**: ~%d tokens\n", f.TokenImpact)
			}
			fmt.Fprintln(w, "")
		}
	}
}
