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
	"github.com/akoserwal/mcpq/pkg/report"
	"github.com/akoserwal/mcpq/pkg/scorer"
	"github.com/spf13/cobra"
)

var (
	analyzeSSEURL   string
	analyzeTracePath string
	analyzeTimeout  time.Duration
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [flags] [-- <server_command> [args...]]",
	Short: "Perform static schema analysis & quality scoring on an MCP server",
	Long: `Connects to an MCP server (via stdio or SSE), fetches all tool definitions,
resources, and prompts, runs static linter rules, and computes the MCP Quality Score (0-100).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), analyzeTimeout)
		defer cancel()

		tok := getTokenizer()
		engine := linter.NewEngine()

		var tools []mcp.Tool
		var resources []mcp.Resource
		var prompts []mcp.Prompt

		// 1. Connect to MCP server if command or SSE provided
		if analyzeSSEURL != "" || len(args) > 0 {
			var client mcp.Client
			var err error

			if analyzeSSEURL != "" {
				client, err = mcp.NewSSEClient(analyzeSSEURL)
				if err != nil {
					return fmt.Errorf("failed to connect to SSE server: %w", err)
				}
			} else {
				targetCmd, targetArgs := parseServerArgs(args)
				if targetCmd == "" {
					return fmt.Errorf("no server command provided. Usage: mcpq analyze -- <command> [args...]")
				}
				client, err = mcp.NewStdioClient(targetCmd, targetArgs, nil)
				if err != nil {
					return fmt.Errorf("failed to start MCP server process: %w", err)
				}
			}
			defer client.Close()

			// Initialize
			initRes, err := client.Initialize(ctx)
			if err != nil {
				return fmt.Errorf("MCP handshake failed: %w", err)
			}

			// List tools
			tools, err = client.ListTools(ctx)
			if err != nil {
				return fmt.Errorf("failed to list tools: %w", err)
			}

			// List resources (if capability declared)
			if initRes != nil && initRes.Capabilities.Resources != nil {
				resources, _ = client.ListResources(ctx)
			}

			// List prompts (if capability declared)
			if initRes != nil && initRes.Capabilities.Prompts != nil {
				prompts, _ = client.ListPrompts(ctx)
			}
		}

		// 2. Load trace data if specified
		var traceEvents []linter.TraceEvent
		if analyzeTracePath != "" {
			events, err := proxy.LoadTraces(analyzeTracePath)
			if err != nil {
				return fmt.Errorf("failed to load trace file: %w", err)
			}
			traceEvents = events
		}

		if len(tools) == 0 && len(traceEvents) == 0 {
			return fmt.Errorf("no tools or traces found to analyze. Please specify a server command, --sse URL, or --trace file")
		}

		// 3. Static & Dynamic Analysis
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

		// 4. Run Linter
		findings := engine.LintAll(staticCtx, dynamicCtx)

		// 5. Calculate Score
		qualityScore := scorer.CalculateScore(schemaAnalysis, traceAnalysis, findings)

		// 6. Output Report
		return outputReport(schemaAnalysis, traceAnalysis, qualityScore)
	},
}

func init() {
	analyzeCmd.Flags().StringVar(&analyzeSSEURL, "sse", "", "Target MCP server SSE URL (e.g. http://localhost:8000/sse)")
	analyzeCmd.Flags().StringVar(&analyzeTracePath, "trace", "", "Path to recorded trace JSON file for dynamic evaluation")
	analyzeCmd.Flags().DurationVar(&analyzeTimeout, "timeout", 30*time.Second, "Timeout for MCP server operations")
	rootCmd.AddCommand(analyzeCmd)
}

func parseServerArgs(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	if args[0] == "stdio" && len(args) > 1 {
		return args[1], args[2:]
	}
	return args[0], args[1:]
}

func outputReport(schema *analyzer.SchemaAnalysis, traces *analyzer.TraceAnalysis, score *scorer.QualityScore) error {
	switch outputFormat {
	case "json":
		return report.RenderJSON(os.Stdout, schema, traces, score, true)
	case "md", "markdown":
		report.RenderMarkdown(os.Stdout, schema, traces, score)
		return nil
	case "sarif":
		return report.RenderSARIF(os.Stdout, score)
	default:
		report.RenderTerminal(os.Stdout, schema, traces, score)
		return nil
	}
}
