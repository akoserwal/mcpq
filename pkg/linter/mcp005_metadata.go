package linter

import (
	"encoding/json"
	"fmt"
	"strings"
)

type RuleMCP005Metadata struct{}

func (r *RuleMCP005Metadata) ID() string {
	return "MCP005"
}

func (r *RuleMCP005Metadata) Name() string {
	return "repeated-metadata"
}

func (r *RuleMCP005Metadata) Severity() Severity {
	return SeverityError
}

func (r *RuleMCP005Metadata) Description() string {
	return "Dynamic responses contain repetitive metadata envelopes (server info, debug traces, echoed configs) that waste context tokens."
}

func (r *RuleMCP005Metadata) CheckDynamic(ctx *DynamicContext) []Finding {
	var findings []Finding

	metadataKeys := []string{
		"server_version", "server_info", "build_version", "environment",
		"trace_id", "request_id", "execution_time_ms", "timestamp",
		"hostname", "node_id", "echo_request", "system_status",
	}

	toolMetadataTokens := make(map[string]int)
	toolTotalTokens := make(map[string]int)
	toolCallCounts := make(map[string]int)

	for _, trace := range ctx.Traces {
		if trace.Response == nil || len(trace.Response.Result) == 0 {
			continue
		}

		toolName := trace.ToolName
		if toolName == "" {
			toolName = trace.Method
		}

		var parsed map[string]any
		if err := json.Unmarshal(trace.Response.Result, &parsed); err != nil {
			continue
		}

		toolCallCounts[toolName]++
		totalTok := trace.TokenCount
		if totalTok == 0 && ctx.Tokenizer != nil {
			totalTok = ctx.Tokenizer.CountTokens(string(trace.Response.Result))
		}
		toolTotalTokens[toolName] += totalTok

		// Check for content object array
		var contentArr []map[string]any
		if cRaw, ok := parsed["content"]; ok {
			if cBytes, err := json.Marshal(cRaw); err == nil {
				_ = json.Unmarshal(cBytes, &contentArr)
			}
		}

		metaTok := 0
		for _, k := range metadataKeys {
			if val, exists := parsed[k]; exists {
				if ctx.Tokenizer != nil {
					metaTok += ctx.Tokenizer.CountJSONTokens(val) + ctx.Tokenizer.CountTokens(k)
				} else {
					metaTok += 20
				}
			}
		}

		// Also check inside text content if JSON
		for _, item := range contentArr {
			if textVal, ok := item["text"].(string); ok && strings.HasPrefix(strings.TrimSpace(textVal), "{") {
				var inner map[string]any
				if err := json.Unmarshal([]byte(textVal), &inner); err == nil {
					for _, k := range metadataKeys {
						if val, exists := inner[k]; exists {
							if ctx.Tokenizer != nil {
								metaTok += ctx.Tokenizer.CountJSONTokens(val) + ctx.Tokenizer.CountTokens(k)
							} else {
								metaTok += 20
							}
						}
					}
				}
			}
		}

		toolMetadataTokens[toolName] += metaTok
	}

	for tool, count := range toolCallCounts {
		total := toolTotalTokens[tool]
		meta := toolMetadataTokens[tool]
		if count >= 2 && total > 0 {
			ratio := float64(meta) / float64(total)
			if ratio > 0.15 && meta > 50 {
				findings = append(findings, Finding{
					RuleID:         r.ID(),
					RuleName:       r.Name(),
					Severity:       r.Severity(),
					TargetType:     "trace",
					TargetName:     tool,
					Message:        fmt.Sprintf("Tool %q responses contain %.1f%% repetitive envelope metadata (%d wasted tokens across %d calls).", tool, ratio*100, meta, count),
					Recommendation: "Strip static server envelopes, timestamps, and echo payloads from tool responses. Move system state to the initial session handshake.",
					TokenImpact:    meta,
				})
			}
		}
	}

	return findings
}
