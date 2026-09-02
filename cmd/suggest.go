package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/suggest"
	"github.com/spf13/cobra"
)

var (
	suggestToolName string
	suggestLang     string
	suggestSSEURL   string
)

var suggestCmd = &cobra.Command{
	Use:   "suggest [flags] [-- <server_command> [args...]]",
	Short: "Generate progressive disclosure refactoring code snippets (TypeScript/Go)",
	Long: `Inspects MCP tool definitions, runs quality diagnostics, and generates
drop-in code refactorings (such as cursor pagination, field selection, summary/detail splits)
to optimize context efficiency.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var client mcp.Client
		var err error

		if suggestSSEURL != "" {
			client, err = mcp.NewSSEClient(suggestSSEURL)
			if err != nil {
				return fmt.Errorf("failed to connect to SSE server: %w", err)
			}
		} else {
			targetCmd, targetArgs := parseServerArgs(args)
			if targetCmd == "" {
				return fmt.Errorf("no server command provided. Usage: mcpq suggest -- <command> [args...]")
			}
			client, err = mcp.NewStdioClient(targetCmd, targetArgs, nil)
			if err != nil {
				return fmt.Errorf("failed to start MCP server: %w", err)
			}
		}
		defer client.Close()

		if _, err := client.Initialize(ctx); err != nil {
			return fmt.Errorf("initialize failed: %w", err)
		}

		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("list tools failed: %w", err)
		}

		tok := getTokenizer()
		engine := linter.NewEngine()
		staticCtx := &linter.StaticContext{
			Tools:     tools,
			Tokenizer: tok,
		}

		findings := engine.LintStatic(staticCtx)
		suggestions := suggest.GenerateSuggestions(tools, findings)

		if len(suggestions) == 0 {
			fmt.Println("✓ No refactoring suggestions needed. Your tools follow progressive disclosure patterns.")
			return nil
		}

		fmt.Printf("\n=== MCPQ Progressive Disclosure Refactoring Suggestions (%d) ===\n\n", len(suggestions))
		for i, s := range suggestions {
			if suggestToolName != "" && s.ToolName != suggestToolName {
				continue
			}

			fmt.Printf("## %d. %s\n", i+1, s.Title)
			fmt.Printf("- **Target Tool**: `%s`\n", s.ToolName)
			fmt.Printf("- **Pattern**: `%s`\n", s.Pattern)
			fmt.Printf("- **Impact**: %s\n", s.EstimatedSaving)
			fmt.Printf("- **Rationale**: %s\n\n", s.Description)

			if (suggestLang == "ts" || suggestLang == "typescript" || suggestLang == "all") && s.TypeScriptCode != "" {
				fmt.Println("### TypeScript Implementation:")
				fmt.Println("```typescript")
				fmt.Println(s.TypeScriptCode)
				fmt.Println("```")
				fmt.Println("")
			}

			if (suggestLang == "go" || suggestLang == "golang" || suggestLang == "all") && s.GoCode != "" {
				fmt.Println("### Go Implementation:")
				fmt.Println("```go")
				fmt.Println(s.GoCode)
				fmt.Println("```")
				fmt.Println("")
			}
		}

		return nil
	},
}

func init() {
	suggestCmd.Flags().StringVar(&suggestToolName, "tool", "", "Filter suggestions for a specific tool name")
	suggestCmd.Flags().StringVar(&suggestLang, "lang", "ts", "Target language code output: ts, go, all")
	suggestCmd.Flags().StringVar(&suggestSSEURL, "sse", "", "Target MCP server SSE URL")
	rootCmd.AddCommand(suggestCmd)
}
