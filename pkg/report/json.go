package report

import (
	"encoding/json"
	"io"

	"github.com/akoserwal/mcpq/pkg/analyzer"
	"github.com/akoserwal/mcpq/pkg/scorer"
)

// JSONReport represents the complete serialized report.
type JSONReport struct {
	QualityScore *scorer.QualityScore     `json:"qualityScore"`
	Schema       *analyzer.SchemaAnalysis `json:"schemaAnalysis,omitempty"`
	Traces       *analyzer.TraceAnalysis  `json:"traceAnalysis,omitempty"`
}

// RenderJSON serializes the analysis and score to JSON.
func RenderJSON(w io.Writer, schema *analyzer.SchemaAnalysis, traces *analyzer.TraceAnalysis, score *scorer.QualityScore, pretty bool) error {
	rep := JSONReport{
		QualityScore: score,
		Schema:       schema,
		Traces:       traces,
	}

	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(rep)
}
