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
