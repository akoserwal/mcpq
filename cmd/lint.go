package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/akoserwal/mcpq/pkg/analyzer"
	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/proxy"
	"github.com/akoserwal/mcpq/pkg/scorer"
	"github.com/spf13/cobra"
)

var (
	lintFailOn    string
	lintMinScore  int
	lintSSEURL    string
	lintTracePath string
	lintTimeout   time.Duration
)

var lintCmd = &cobra.Command{
	Use:   "lint [flags] [-- <server_command> [args...]]",
	Short: "Audit an MCP server with anti-pattern rules (MCP001-MCP010) for CI/CD",
	Long: `Audits MCP schemas and optional traces against efficiency rules (MCP001 - MCP010).
Exits with code 1 if violations match or exceed the specified fail threshold or score falls below --min-score.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), lintTimeout)
		defer cancel()

		tok := getTokenizer()
		engine := linter.NewEngine()

		var tools []mcp.Tool
		var resources []mcp.Resource
		var prompts []mcp.Prompt

		if lintSSEURL != "" || len(args) > 0 {
			var client mcp.Client
			var err error

			if lintSSEURL != "" {
				client, err = mcp.NewSSEClient(lintSSEURL)
				if err != nil {
					return fmt.Errorf("failed to connect to SSE server: %w", err)
				}
			} else {
				targetCmd, targetArgs := parseServerArgs(args)
				if targetCmd == "" {
					return fmt.Errorf("no server command provided. Usage: mcpq lint -- <command> [args...]")
				}
				client, err = mcp.NewStdioClient(targetCmd, targetArgs, nil)
				if err != nil {
					return fmt.Errorf("failed to start MCP server process: %w", err)
				}
			}
			defer client.Close()

			initRes, err := client.Initialize(ctx)
			if err != nil {
				return fmt.Errorf("MCP handshake failed: %w", err)
			}

			tools, err = client.ListTools(ctx)
			if err != nil {
				return fmt.Errorf("failed to list tools: %w", err)
			}
			if initRes != nil && initRes.Capabilities.Resources != nil {
				resources, _ = client.ListResources(ctx)
			}
			if initRes != nil && initRes.Capabilities.Prompts != nil {
				prompts, _ = client.ListPrompts(ctx)
			}
		}

		var traceEvents []linter.TraceEvent
		if lintTracePath != "" {
			events, err := proxy.LoadTraces(lintTracePath)
			if err != nil {
				return fmt.Errorf("failed to load trace file: %w", err)
			}
			traceEvents = events
		}

		if len(tools) == 0 && len(traceEvents) == 0 {
			return fmt.Errorf("no tools or traces found to lint")
		}

		var schemaAnalysis *analyzer.SchemaAnalysis
		var staticCtx *linter.StaticContext
		if len(tools) > 0 {
			schemaAnalysis = analyzer.AnalyzeSchema(tools, resources, prompts, tok)
			staticCtx = &linter.StaticContext{
				Tools:     tools,
				Resources: resources,
				Prompts:   prompts,
				Tokenizer: tok,
			}
		}

		var traceAnalysis *analyzer.TraceAnalysis
		var dynamicCtx *linter.DynamicContext
		if len(traceEvents) > 0 {
			traceAnalysis = analyzer.AnalyzeTraces(traceEvents, tok)
			dynamicCtx = &linter.DynamicContext{
				Traces:    traceEvents,
				Tokenizer: tok,
			}
		}

		findings := engine.LintAll(staticCtx, dynamicCtx)
		qualityScore := scorer.CalculateScore(schemaAnalysis, traceAnalysis, findings)

		if err := outputReport(schemaAnalysis, traceAnalysis, qualityScore); err != nil {
			return err
		}

		// Check failure conditions
		hasFailures := false
		for _, f := range findings {
			switch lintFailOn {
			case "error":
				if f.Severity == linter.SeverityError {
					hasFailures = true
				}
			case "warning":
				if f.Severity == linter.SeverityError || f.Severity == linter.SeverityWarning {
					hasFailures = true
				}
			case "info":
				hasFailures = true
			}
		}

		if lintMinScore > 0 && qualityScore.Score < lintMinScore {
			hasFailures = true
		}

		if hasFailures {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	lintCmd.Flags().StringVar(&lintFailOn, "fail-on", "error", "Fail threshold: error, warning, info, none")
	lintCmd.Flags().IntVar(&lintMinScore, "min-score", 0, "Minimum required score (0-100) to pass linting")
	lintCmd.Flags().StringVar(&lintSSEURL, "sse", "", "Target MCP server SSE URL")
	lintCmd.Flags().StringVar(&lintTracePath, "trace", "", "Path to trace JSON file")
	lintCmd.Flags().DurationVar(&lintTimeout, "timeout", 30*time.Second, "MCP server handshake timeout")
	rootCmd.AddCommand(lintCmd)
}
