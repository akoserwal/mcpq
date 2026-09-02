package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/akoserwal/mcpq/pkg/proxy"
	"github.com/spf13/cobra"
)

var (
	recordOutputPath string
)

var recordCmd = &cobra.Command{
	Use:   "record [flags] -o <output_file.json> -- <server_command> [args...]",
	Short: "Record live MCP JSON-RPC traffic to a trace file",
	Long: `Spawns the target MCP server subprocess and acts as a transparent stdio proxy.
All JSON-RPC requests and responses between the client (e.g. Claude Desktop, Cursor)
and the MCP server are timestamped, measured, and saved to a trace JSON file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if recordOutputPath == "" {
			return fmt.Errorf("output file path is required. Specify with -o <traces.json>")
		}

		targetCmd, targetArgs := parseServerArgs(args)
		if targetCmd == "" {
			return fmt.Errorf("no server command provided. Usage: mcpq record -o traces.json -- <command> [args...]")
		}

		tok := getTokenizer()
		stdioProxy := proxy.NewStdioProxy(targetCmd, targetArgs, tok)

		// Handle clean shutdown on Ctrl+C / SIGTERM
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			_ = stdioProxy.Recorder().SaveToFile(recordOutputPath)
			os.Exit(0)
		}()

		fmt.Fprintf(os.Stderr, "mcpq: Recording live MCP traffic to %s...\n", recordOutputPath)
		err := stdioProxy.RunInteractive()

		// Save recorded traces on process exit
		saveErr := stdioProxy.Recorder().SaveToFile(recordOutputPath)
		if saveErr != nil {
			fmt.Fprintf(os.Stderr, "mcpq: failed to save traces: %v\n", saveErr)
		} else {
			events := stdioProxy.Recorder().Events()
			fmt.Fprintf(os.Stderr, "mcpq: Successfully recorded %d events to %s\n", len(events), recordOutputPath)
		}

		return err
	},
}

func init() {
	recordCmd.Flags().StringVarP(&recordOutputPath, "output", "o", "traces.json", "Output trace JSON file path")
	rootCmd.AddCommand(recordCmd)
}
