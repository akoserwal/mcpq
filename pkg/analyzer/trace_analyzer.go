package analyzer

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/tokenizer"
)

// ToolTraceStats holds per-tool dynamic runtime metrics.
type ToolTraceStats struct {
	ToolName    string  `json:"toolName"`
	CallCount   int     `json:"callCount"`
	TotalTokens int     `json:"totalTokens"`
	AvgTokens   float64 `json:"avgTokens"`
	P50Tokens   int     `json:"p50Tokens"`
	P95Tokens   int     `json:"p95Tokens"`
	MaxTokens   int     `json:"maxTokens"`
}

// TraceAnalysis holds aggregated metrics across dynamic traces.
type TraceAnalysis struct {
	TotalEvents        int                       `json:"totalEvents"`
	TotalToolCalls     int                       `json:"totalToolCalls"`
	TotalRequestTokens int                       `json:"totalRequestTokens"`
	TotalRespTokens    int                       `json:"totalResponseTokens"`
	TotalTokens        int                       `json:"totalTokens"`
	P50Tokens          int                       `json:"p50Tokens"`
	P90Tokens          int                       `json:"p90Tokens"`
	P95Tokens          int                       `json:"p95Tokens"`
	P99Tokens          int                       `json:"p99Tokens"`
	MinTokens          int                       `json:"minTokens"`
	MaxTokens          int                       `json:"maxTokens"`
	AvgDurationMs      float64                   `json:"avgDurationMs"`
	P95DurationMs      int64                     `json:"p95DurationMs"`
	MetadataWasteRatio float64                   `json:"metadataWasteRatio"` // 0.0 to 1.0
	Tools              map[string]ToolTraceStats `json:"tools"`
}

// AnalyzeTraces computes statistical token distributions and waste metrics from traces.
func AnalyzeTraces(traces []linter.TraceEvent, tok tokenizer.Tokenizer) *TraceAnalysis {
	if tok == nil {
		tok = tokenizer.DefaultTokenizer()
	}

	analysis := &TraceAnalysis{
		TotalEvents: len(traces),
		MinTokens:   math.MaxInt32,
		Tools:       make(map[string]ToolTraceStats),
	}

	if len(traces) == 0 {
		analysis.MinTokens = 0
		return analysis
	}

	var respTokenSizes []int
	var durations []int64
	var totalWasteTokens int

	toolTokensMap := make(map[string][]int)
	toolDurationsMap := make(map[string][]int64)

	for _, trace := range traces {
		// Count request tokens
		if trace.Request != nil {
			reqBytes, _ := json.Marshal(trace.Request)
			reqTok := tok.CountTokens(string(reqBytes))
			analysis.TotalRequestTokens += reqTok
		}

		// Count response tokens
		if trace.Response != nil {
			respBytes, _ := json.Marshal(trace.Response)
			respTok := trace.TokenCount
			if respTok == 0 {
				respTok = tok.CountTokens(string(respBytes))
			}
			analysis.TotalRespTokens += respTok
			respTokenSizes = append(respTokenSizes, respTok)

			if trace.DurationMs > 0 {
				durations = append(durations, trace.DurationMs)
			}

			toolName := trace.ToolName
			if toolName == "" {
				toolName = trace.Method
			}
			if toolName != "" {
				toolTokensMap[toolName] = append(toolTokensMap[toolName], respTok)
				if trace.DurationMs > 0 {
					toolDurationsMap[toolName] = append(toolDurationsMap[toolName], trace.DurationMs)
				}
			}

			// Estimate static envelope waste
			totalWasteTokens += estimateEnvelopeWaste(trace.Response.Result, tok)
		}
	}

	analysis.TotalTokens = analysis.TotalRequestTokens + analysis.TotalRespTokens
	analysis.TotalToolCalls = len(respTokenSizes)

	if len(respTokenSizes) > 0 {
		sort.Ints(respTokenSizes)
		analysis.MinTokens = respTokenSizes[0]
		analysis.MaxTokens = respTokenSizes[len(respTokenSizes)-1]
		analysis.P50Tokens = percentileInt(respTokenSizes, 0.50)
		analysis.P90Tokens = percentileInt(respTokenSizes, 0.90)
		analysis.P95Tokens = percentileInt(respTokenSizes, 0.95)
		analysis.P99Tokens = percentileInt(respTokenSizes, 0.99)
	} else {
		analysis.MinTokens = 0
	}

	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		var sumDur int64
		for _, d := range durations {
			sumDur += d
		}
		analysis.AvgDurationMs = float64(sumDur) / float64(len(durations))
		analysis.P95DurationMs = percentileInt64(durations, 0.95)
	}

	if analysis.TotalRespTokens > 0 {
		analysis.MetadataWasteRatio = float64(totalWasteTokens) / float64(analysis.TotalRespTokens)
		if analysis.MetadataWasteRatio > 1.0 {
			analysis.MetadataWasteRatio = 1.0
		}
	}

	// Tool breakdowns
	for toolName, sizes := range toolTokensMap {
		sort.Ints(sizes)
		var total int
		for _, s := range sizes {
			total += s
		}
		avg := float64(total) / float64(len(sizes))
		p50 := percentileInt(sizes, 0.50)
		p95 := percentileInt(sizes, 0.95)
		maxTok := sizes[len(sizes)-1]

		analysis.Tools[toolName] = ToolTraceStats{
			ToolName:    toolName,
			CallCount:   len(sizes),
			TotalTokens: total,
			AvgTokens:   avg,
			P50Tokens:   p50,
			P95Tokens:   p95,
			MaxTokens:   maxTok,
		}
	}

	return analysis
}

func estimateEnvelopeWaste(rawResult json.RawMessage, tok tokenizer.Tokenizer) int {
	if len(rawResult) == 0 {
		return 0
	}
	var obj map[string]any
	if err := json.Unmarshal(rawResult, &obj); err != nil {
		return 0
	}

	wasteKeys := []string{"server_version", "server_info", "timestamp", "trace_id", "execution_time_ms", "hostname", "build_version"}
	wasteTokens := 0
	for _, k := range wasteKeys {
		if val, exists := obj[k]; exists {
			wasteTokens += tok.CountJSONTokens(val) + tok.CountTokens(k)
		}
	}

	// Look in content array text
	if contentRaw, exists := obj["content"]; exists {
		var contentArr []map[string]any
		if cBytes, err := json.Marshal(contentRaw); err == nil {
			if err := json.Unmarshal(cBytes, &contentArr); err == nil {
				for _, item := range contentArr {
					if textVal, ok := item["text"].(string); ok && strings.HasPrefix(strings.TrimSpace(textVal), "{") {
						var inner map[string]any
						if err := json.Unmarshal([]byte(textVal), &inner); err == nil {
							for _, k := range wasteKeys {
								if val, exists := inner[k]; exists {
									wasteTokens += tok.CountJSONTokens(val) + tok.CountTokens(k)
								}
							}
						}
					}
				}
			}
		}
	}

	return wasteTokens
}

func percentileInt(arr []int, p float64) int {
	if len(arr) == 0 {
		return 0
	}
	idx := int(math.Floor(float64(len(arr)-1) * p))
	return arr[idx]
}

func percentileInt64(arr []int64, p float64) int64 {
	if len(arr) == 0 {
		return 0
	}
	idx := int(math.Floor(float64(len(arr)-1) * p))
	return arr[idx]
}
