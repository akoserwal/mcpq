package cmd

import (
	"fmt"
	"os"

	"github.com/akoserwal/mcpq/pkg/tokenizer"
	"github.com/spf13/cobra"
)

var (
	outputFormat  string
	tokenizerName string
	verbose       bool
)

var rootCmd = &cobra.Command{
	Use:   "mcpq",
	Short: "mcpq — MCP Quality & Context Efficiency Analyzer",
	Long: `mcpq is a developer tool, linter, proxy, and benchmark suite designed to measure,
audit, and optimize Model Context Protocol (MCP) server context consumption, tool design modularity,
and progressive disclosure efficiency.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "term", "Output format: term, json, md, sarif")
	rootCmd.PersistentFlags().StringVar(&tokenizerName, "tokenizer", "cl100k_base", "Tokenizer encoding: cl100k_base, o200k_base, fast")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug output")
}

func getTokenizer() tokenizer.Tokenizer {
	if tokenizerName == "fast" {
		return tokenizer.NewFastTokenizer()
	}
	t, err := tokenizer.NewTiktokenTokenizer(tokenizerName)
	if err != nil {
		return tokenizer.NewFastTokenizer()
	}
	return t
}
