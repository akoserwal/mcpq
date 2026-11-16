package scorer

import (
	"github.com/akoserwal/mcpq/pkg/linter"
)

// ScoreBreakdown details the individual sub-scores.
type ScoreBreakdown struct {
	SchemaEfficiency      float64 `json:"schemaEfficiency"`      // max 20
	ToolDesign            float64 `json:"toolDesign"`            // max 20
	ResponseEfficiency    float64 `json:"responseEfficiency"`    // max 30
	ProgressiveDisclosure float64 `json:"progressiveDisclosure"` // max 15
	Discoverability       float64 `json:"discoverability"`       // max 10
	Penalties             float64 `json:"penalties"`             // subtracted
}

// Grade represents the quality rating grade.
type Grade string

const (
	GradeAPlus Grade = "A+"
	GradeA     Grade = "A"
	GradeB     Grade = "B"
	GradeC     Grade = "C"
	GradeD     Grade = "D"
	GradeF     Grade = "F"
)

// QualityScore contains overall quality score and rating metrics.
type QualityScore struct {
	Scope          string           `json:"scope"` // static, runtime, combined, or none
	MeasuredPoints float64          `json:"measuredPoints"`
	RuntimeSamples int              `json:"runtimeSamples"`
	Score          int              `json:"score"` // 0 to 100
	Grade          Grade            `json:"grade"`
	Summary        string           `json:"summary"`
	Breakdown      ScoreBreakdown   `json:"breakdown"`
	Findings       []linter.Finding `json:"findings"`
}

// ComputeGrade maps a numerical score to a grade.
func ComputeGrade(score int) Grade {
	switch {
	case score >= 95:
		return GradeAPlus
	case score >= 90:
		return GradeA
	case score >= 80:
		return GradeB
	case score >= 70:
		return GradeC
	case score >= 60:
		return GradeD
	default:
		return GradeF
	}
}

// GetGradeSummary returns a human-readable interpretation of the grade.
func GetGradeSummary(grade Grade) string {
	switch grade {
	case GradeAPlus:
		return "Exceptional MCP Design. Optimal token efficiency and progressive disclosure."
	case GradeA:
		return "Excellent context efficiency with clean schema modularity."
	case GradeB:
		return "Good MCP architecture. Minor token optimization opportunities detected."
	case GradeC:
		return "Fair. Noticeable token bloat and lack of progressive disclosure controls."
	case GradeD:
		return "Poor context efficiency. Critical anti-patterns present that bloat LLM context."
	default:
		return "Failing context efficiency. Monolithic tools or unbounded payloads present."
	}
}
