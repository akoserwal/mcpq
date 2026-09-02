package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akoserwal/mcpq/pkg/proxy"
	"github.com/spf13/cobra"
)

var (
	proxyListenAddr string
	proxyTargetURL  string
	proxyOutputFile string
)

var proxyCmd = &cobra.Command{
	Use:   "proxy [flags] [-- <server_command> [args...]]",
	Short: "Run an MCP proxy interceptor (SSE or stdio) to monitor & record traffic",
	Long: `Starts a proxy server that intercepts and logs JSON-RPC traffic for Model Context Protocol.
Supports reverse proxying SSE endpoints as well as wrapping stdio subprocesses.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tok := getTokenizer()

		if proxyTargetURL != "" {
			// SSE Reverse Proxy Mode
			sseProxy, err := proxy.NewSSEProxy(proxyTargetURL, tok)
			if err != nil {
				return fmt.Errorf("failed to create SSE proxy: %w", err)
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				if proxyOutputFile != "" {
					_ = sseProxy.Recorder().SaveToFile(proxyOutputFile)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = sseProxy.Shutdown(ctx)
				os.Exit(0)
			}()

			fmt.Fprintf(os.Stderr, "mcpq: Starting SSE reverse proxy on %s -> %s\n", proxyListenAddr, proxyTargetURL)
			return sseProxy.Start(proxyListenAddr)
		}

		// Stdio Mode
		targetCmd, targetArgs := parseServerArgs(args)
		if targetCmd == "" {
			return fmt.Errorf("must specify either --target <SSE_URL> or a subprocess command to proxy")
		}

		stdioProxy := proxy.NewStdioProxy(targetCmd, targetArgs, tok)
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			if proxyOutputFile != "" {
				_ = stdioProxy.Recorder().SaveToFile(proxyOutputFile)
			}
			os.Exit(0)
		}()

		err := stdioProxy.RunInteractive()
		if proxyOutputFile != "" {
			_ = stdioProxy.Recorder().SaveToFile(proxyOutputFile)
		}
		return err
	},
}

func init() {
	proxyCmd.Flags().StringVar(&proxyListenAddr, "listen", ":8080", "HTTP listen address for SSE proxy")
	proxyCmd.Flags().StringVar(&proxyTargetURL, "target", "", "Target SSE URL to reverse proxy")
	proxyCmd.Flags().StringVarP(&proxyOutputFile, "output", "o", "proxy_traces.json", "File path to save recorded traces")
	rootCmd.AddCommand(proxyCmd)
}
