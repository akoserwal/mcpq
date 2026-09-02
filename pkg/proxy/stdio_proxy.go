package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

// StdioProxy wraps an MCP server subprocess and intercepts stdio traffic.
type StdioProxy struct {
	serverCmd  string
	serverArgs []string
	recorder   *Recorder
	tok        tokenizer.Tokenizer
}

// NewStdioProxy creates a StdioProxy for the given command.
func NewStdioProxy(serverCmd string, serverArgs []string, tok tokenizer.Tokenizer) *StdioProxy {
	if tok == nil {
		tok = tokenizer.DefaultTokenizer()
	}
	fullCmd := serverCmd
	if len(serverArgs) > 0 {
		fullCmd = fmt.Sprintf("%s %s", serverCmd, fmt.Sprint(serverArgs))
	}

	return &StdioProxy{
		serverCmd:  serverCmd,
		serverArgs: serverArgs,
		recorder:   NewRecorder(fullCmd, tok),
		tok:        tok,
	}
}

// Recorder returns the attached trace recorder.
func (p *StdioProxy) Recorder() *Recorder {
	return p.recorder
}

// Run executes the server and intercepts in/out streams until exit.
func (p *StdioProxy) Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.Command(p.serverCmd, p.serverArgs...)

	serverIn, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open server stdin pipe: %w", err)
	}
	defer serverIn.Close()

	serverOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open server stdout pipe: %w", err)
	}
	defer serverOut.Close()

	serverErr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to open server stderr pipe: %w", err)
	}
	defer serverErr.Close()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server subprocess %q: %w", p.serverCmd, err)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// Forward Stderr
	go func() {
		defer wg.Done()
		if stderr != nil {
			_, _ = io.Copy(stderr, serverErr)
		}
	}()

	// Intercept Stdin (Client -> Server)
	go func() {
		defer wg.Done()
		defer serverIn.Close()

		scanner := bufio.NewScanner(stdin)
		const maxBuf = 16 * 1024 * 1024
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, maxBuf)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// Record if valid JSON-RPC request
			var req mcp.Request
			if err := json.Unmarshal(line, &req); err == nil && req.JSONRPC != "" {
				p.recorder.RecordRequest(&req)
			}

			// Forward to server
			lineWithNewline := append(bytes.Clone(line), '\n')
			if _, err := serverIn.Write(lineWithNewline); err != nil {
				break
			}
		}
	}()

	// Intercept Stdout (Server -> Client)
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(serverOut)
		const maxBuf = 16 * 1024 * 1024
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, maxBuf)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// Record if valid JSON-RPC response
			var resp mcp.Response
			if err := json.Unmarshal(line, &resp); err == nil && resp.JSONRPC != "" {
				p.recorder.RecordResponse(&resp)
			}

			// Forward to client stdout
			lineWithNewline := append(bytes.Clone(line), '\n')
			if _, err := stdout.Write(lineWithNewline); err != nil {
				break
			}
		}
	}()

	err = cmd.Wait()
	wg.Wait()
	return err
}

// RunInteractive runs proxy on standard OS stdin/stdout.
func (p *StdioProxy) RunInteractive() error {
	return p.Run(os.Stdin, os.Stdout, os.Stderr)
}
