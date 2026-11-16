package scorer

import (
	"testing"

	"github.com/akoserwal/mcpq/pkg/analyzer"
	"github.com/akoserwal/mcpq/pkg/linter"
)

func TestCalculateScore(t *testing.T) {
	// 1. Test clean schema (should receive high score / Grade A+)
	cleanSchema := &analyzer.SchemaAnalysis{
		ToolCount:        3,
		TotalTokens:      300,
		AvgTokensPerTool: 100,
		TotalParams:      6,
		AvgParamsPerTool: 2.0,
		ToolStats: []analyzer.ToolSchemaStats{
			{Name: "get_user", Description: "Get user by ID", DescriptionLen: 14},
			{Name: "list_users", Description: "List users with pagination", DescriptionLen: 26},
		},
	}

	score := CalculateScore(cleanSchema, nil, nil)
	if score.Score < 90 {
		t.Errorf("expected score >= 90 for clean schema, got %d", score.Score)
	}
	if score.Grade != GradeAPlus && score.Grade != GradeA {
		t.Errorf("expected Grade A/A+, got %s", score.Grade)
	}

	// 2. Test bad schema with critical error findings
	badFindings := []linter.Finding{
		{RuleID: "MCP001", Severity: linter.SeverityError},
		{RuleID: "MCP002", Severity: linter.SeverityError},
		{RuleID: "MCP003", Severity: linter.SeverityError},
		{RuleID: "MCP004", Severity: linter.SeverityWarning},
		{RuleID: "MCP007", Severity: linter.SeverityWarning},
	}

	badScore := CalculateScore(cleanSchema, nil, badFindings)
	if badScore.Score >= 70 {
		t.Errorf("expected low score for failing server, got %d", badScore.Score)
	}
	if badScore.Grade != GradeD && badScore.Grade != GradeF {
		t.Errorf("expected Grade D or F, got %s", badScore.Grade)
	}
}

func TestScoreUsesOnlyMeasuredDimensions(t *testing.T) {
	schema := &analyzer.SchemaAnalysis{ToolCount: 1, TotalTokens: 100, AvgTokensPerTool: 100, ToolStats: []analyzer.ToolSchemaStats{{DescriptionLen: 20}}}
	traces := &analyzer.TraceAnalysis{TotalToolCalls: 1, P50Tokens: 20, P95Tokens: 20}
	for _, tc := range []struct {
		name   string
		schema *analyzer.SchemaAnalysis
		traces *analyzer.TraceAnalysis
		points float64
	}{
		{"static", schema, nil, 65}, {"runtime", nil, traces, 30}, {"combined", schema, traces, 95}, {"none", nil, nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			score := CalculateScore(tc.schema, tc.traces, nil)
			if score.MeasuredPoints != tc.points || score.Scope != tc.name {
				t.Fatalf("incorrect evidence: %+v", score)
			}
			expected := 100
			if tc.points == 0 {
				expected = 0
			}
			if score.Score != expected {
				t.Fatalf("score=%d want %d", score.Score, expected)
			}
			if tc.traces == nil && score.Breakdown.ResponseEfficiency != 0 {
				t.Fatal("unmeasured runtime received points")
			}
			if tc.schema == nil && (score.Breakdown.SchemaEfficiency != 0 || score.Breakdown.ProgressiveDisclosure != 0) {
				t.Fatal("unmeasured schema received points")
			}
		})
	}
}
