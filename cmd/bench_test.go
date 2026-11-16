package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/akoserwal/mcpq/pkg/mcp"
	"github.com/akoserwal/mcpq/pkg/report"
)

func TestBenchmarkCapturesDynamicResponses(t *testing.T) {
	if os.Getenv("MCPQ_BENCH_HELPER") == "1" {
		decoder, encoder := json.NewDecoder(os.Stdin), json.NewEncoder(os.Stdout)
		for {
			var req mcp.Request
			if decoder.Decode(&req) != nil {
				os.Exit(0)
			}
			if req.ID == nil {
				continue
			}
			var result any
			switch req.Method {
			case "initialize":
				result = map[string]any{"protocolVersion": mcp.LatestProtocolVersion, "capabilities": map[string]any{}, "serverInfo": map[string]string{"name": "test", "version": "1"}}
			case "tools/list":
				result = map[string]any{"tools": []map[string]any{{"name": "test", "description": "Test benchmark response capture", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}}}}
			case "tools/call":
				time.Sleep(5 * time.Millisecond)
				text, _ := json.Marshal(map[string]string{"server_info": strings.Repeat("metadata ", 10000)})
				result = mcp.CallToolResult{Content: []mcp.Content{{Type: "text", Text: string(text)}}}
				switch os.Getenv("MCPQ_BENCH_MODE") {
				case "tool_error":
					result = mcp.CallToolResult{IsError: true, Content: []mcp.Content{{Type: "text", Text: "conversion failed"}}}
				case "rpc_error":
					encoder.Encode(mcp.Response{JSONRPC: "2.0", ID: req.ID, Error: &mcp.RPCError{Code: -32602, Message: "invalid arguments"}})
					continue
				case "structured":
					result = map[string]any{"content": []map[string]string{{"type": "text", "text": "short"}}, "structuredContent": map[string]string{"result": strings.Repeat("large payload ", 10000)}}
				}
			default:
				os.Exit(2)
			}
			data, _ := json.Marshal(result)
			if encoder.Encode(mcp.Response{JSONRPC: "2.0", ID: req.ID, Result: data}) != nil {
				os.Exit(3)
			}
		}
	}

	rep, err := runBenchmarkHelper(t, "")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Traces == nil || rep.Traces.TotalToolCalls != 2 || rep.Traces.P95Tokens <= 8000 || rep.Traces.AvgDurationMs <= 0 {
		t.Fatalf("missing metrics: %+v", rep.Traces)
	}
	found := map[string]bool{}
	for _, finding := range rep.QualityScore.Findings {
		found[finding.RuleID] = true
	}
	if !found["MCP005"] || !found["MCP010"] {
		t.Fatalf("missing dynamic findings: %v", found)
	}
}

func runBenchmarkHelper(t *testing.T, mode string) (*report.JSONReport, error) {
	t.Helper()
	t.Setenv("MCPQ_BENCH_HELPER", "1")
	t.Setenv("MCPQ_BENCH_MODE", mode)
	oldFormat, oldTokenizer := outputFormat, tokenizerName
	outputFormat, tokenizerName = "json", "fast"
	defer func() { outputFormat, tokenizerName = oldFormat, oldTokenizer }()
	out, err := os.CreateTemp(t.TempDir(), "bench")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	oldStdout := os.Stdout
	os.Stdout = out
	defer func() { os.Stdout = oldStdout }()
	if err := benchCmd.RunE(benchCmd, []string{os.Args[0], "-test.run=^TestBenchmarkCapturesDynamicResponses$"}); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	var rep report.JSONReport
	if err := json.Unmarshal(data, &rep); err != nil {
		t.Fatal(err)
	}

	return &rep, nil
}

func TestBenchmarkRejectsFailedCalls(t *testing.T) {
	for _, mode := range []string{"tool_error", "rpc_error"} {
		t.Run(mode, func(t *testing.T) {
			_, err := runBenchmarkHelper(t, mode)
			if err == nil || !strings.Contains(err.Error(), "iteration 1") {
				t.Fatalf("expected scenario failure, got %v", err)
			}
		})
	}
}

func TestBenchmarkCountsStructuredContent(t *testing.T) {
	rep, err := runBenchmarkHelper(t, "structured")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Traces.MaxTokens <= 8000 {
		t.Fatalf("structured content missing from tokens: %+v", rep.Traces)
	}
	for _, finding := range rep.QualityScore.Findings {
		if finding.RuleID == "MCP010" {
			return
		}
	}
	t.Fatal("large structured payload was not flagged")
}
