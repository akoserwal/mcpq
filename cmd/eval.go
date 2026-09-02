package cmd

import (
	"fmt"

	"github.com/akoserwal/mcpq/pkg/analyzer"
	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/proxy"
	"github.com/akoserwal/mcpq/pkg/scorer"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval <trace_file.json>",
	Short: "Evaluate recorded MCP runtime traces for context bloat and token distributions",
	Long: `Loads recorded JSON-RPC traces, computes P50/P90/P95/P99 token distributions,
measures repetitive static envelope waste, runs dynamic anti-pattern rules (MCP005, MCP010),
and computes the dynamic efficiency score.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tracePath := args[0]
		events, err := proxy.LoadTraces(tracePath)
		if err != nil {
			return fmt.Errorf("failed to load trace file: %w", err)
		}

		tok := getTokenizer()
		traceAnalysis := analyzer.AnalyzeTraces(events, tok)

		engine := linter.NewEngine()
		dynamicCtx := &linter.DynamicContext{
			Traces:    events,
			Tokenizer: tok,
		}

		findings := engine.LintDynamic(dynamicCtx)
		qualityScore := scorer.CalculateScore(nil, traceAnalysis, findings)

		return outputReport(nil, traceAnalysis, qualityScore)
	},
}

func init() {
	rootCmd.AddCommand(evalCmd)
}
