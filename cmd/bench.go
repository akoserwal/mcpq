package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/akoserwal/mcpq/pkg/analyzer"
	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/scorer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	benchConfigFile string
	benchSSEURL     string
	benchTimeout    time.Duration
)

// BenchConfig defines benchmark scenarios in YAML.
type BenchConfig struct {
	Scenarios []Scenario `yaml:"scenarios"`
}

type Scenario struct {
	Name       string         `yaml:"name"`
	Tool       string         `yaml:"tool"`
	Args       map[string]any `yaml:"args"`
	Iterations int            `yaml:"iterations"`
}

var benchCmd = &cobra.Command{
	Use:   "bench [flags] [-- <server_command> [args...]]",
	Short: "Benchmark MCP server performance, context consumption, and latency",
	Long: `Executes automated benchmark scenarios against an MCP server,
capturing runtime tokens, latency distributions, and memory payload sizes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), benchTimeout)
		defer cancel()

		var client mcp.Client
		var err error

		if benchSSEURL != "" {
			client, err = mcp.NewSSEClient(benchSSEURL)
			if err != nil {
				return fmt.Errorf("failed to connect to SSE server: %w", err)
			}
		} else {
			targetCmd, targetArgs := parseServerArgs(args)
			if targetCmd == "" {
				return fmt.Errorf("no server command provided. Usage: mcpq bench -- <command> [args...]")
			}
			client, err = mcp.NewStdioClient(targetCmd, targetArgs, nil)
			if err != nil {
				return fmt.Errorf("failed to start MCP server: %w", err)
			}
		}
		defer client.Close()

		tok := getTokenizer()
		var traceEvents []linter.TraceEvent

		// Handshake
		initStart := time.Now()
		initRes, err := client.Initialize(ctx)
		if err != nil {
			return fmt.Errorf("initialize failed: %w", err)
		}
		initDur := time.Since(initStart).Milliseconds()
		fmt.Fprintf(os.Stderr, "✓ MCP Handshake completed with %s v%s in %dms\n", initRes.ServerInfo.Name, initRes.ServerInfo.Version, initDur)

		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("list tools failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Discovered %d tools\n\n", len(tools))

		var scenarios []Scenario
		if benchConfigFile != "" {
			cfgBytes, err := os.ReadFile(benchConfigFile)
			if err != nil {
				return fmt.Errorf("failed to read bench config: %w", err)
			}
			var cfg BenchConfig
			if err := yaml.Unmarshal(cfgBytes, &cfg); err != nil {
				return fmt.Errorf("failed to parse bench config YAML: %w", err)
			}
			scenarios = cfg.Scenarios
		} else {
			// Auto-generate scenarios from available tools
			for _, t := range tools {
				scenarios = append(scenarios, Scenario{
					Name:       fmt.Sprintf("call_%s", t.Name),
					Tool:       t.Name,
					Args:       make(map[string]any),
					Iterations: 2,
				})
			}
		}

		fmt.Fprintln(os.Stderr, "Running benchmark scenarios...")
		for _, sc := range scenarios {
			iters := sc.Iterations
			if iters <= 0 {
				iters = 1
			}

			fmt.Fprintf(os.Stderr, "  • Running scenario %q (tool: %s, iterations: %d)... ", sc.Name, sc.Tool, iters)
			for i := 0; i < iters; i++ {
				callStart := time.Now()
				res, err := client.CallTool(ctx, sc.Tool, sc.Args)
				callDur := time.Since(callStart).Milliseconds()

				if err != nil {
					return fmt.Errorf("scenario %q iteration %d: %w", sc.Name, i+1, err)
				}

				if res.IsError {
					return fmt.Errorf("scenario %q iteration %d: tool %q returned isError=true", sc.Name, i+1, sc.Tool)
				}
				result := res.RawJSON
				if len(result) == 0 {
					result, err = json.Marshal(res)
					if err != nil {
						return fmt.Errorf("encode benchmark response: %w", err)
					}
				}
				response := &mcp.Response{JSONRPC: mcp.JSONRPCVersion, Result: result}

				traceEvents = append(traceEvents, linter.TraceEvent{
					Timestamp:  time.Now(),
					Direction:  "response",
					Method:     "tools/call",
					ToolName:   sc.Tool,
					DurationMs: callDur,
					Response:   response,
					TokenCount: tok.CountJSONTokens(response),
				})
			}
			fmt.Fprintf(os.Stderr, "done.\n")
		}

		fmt.Fprintln(os.Stderr)

		// Analyze
		schemaAnalysis := analyzer.AnalyzeSchema(tools, nil, nil, tok)
		traceAnalysis := analyzer.AnalyzeTraces(traceEvents, tok)
		engine := linter.NewEngine()
		findings := engine.LintAll(&linter.StaticContext{Tools: tools, Tokenizer: tok}, &linter.DynamicContext{Traces: traceEvents, Tokenizer: tok})
		qualityScore := scorer.CalculateScore(schemaAnalysis, traceAnalysis, findings)

		return outputReport(schemaAnalysis, traceAnalysis, qualityScore)
	},
}

func init() {
	benchCmd.Flags().StringVarP(&benchConfigFile, "config", "c", "", "Path to YAML benchmark configuration")
	benchCmd.Flags().StringVar(&benchSSEURL, "sse", "", "Target MCP server SSE URL")
	benchCmd.Flags().DurationVar(&benchTimeout, "timeout", 60*time.Second, "Benchmark execution timeout")
	rootCmd.AddCommand(benchCmd)
}
