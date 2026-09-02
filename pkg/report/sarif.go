package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/akoserwal/mcpq/pkg/linter"
	"github.com/akoserwal/mcpq/pkg/scorer"
)

// SARIFReport represents OASIS SARIF 2.1.0 standard schema.
type SARIFReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []SARIFRun `json:"runs"`
}

type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []SARIFRule `json:"rules"`
}

type SARIFRule struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	ShortDescription SARIFMessageText `json:"shortDescription"`
}

type SARIFResult struct {
	RuleID    string           `json:"ruleId"`
	Level     string           `json:"level"` // "error", "warning", "note"
	Message   SARIFMessageText `json:"message"`
	Locations []SARIFLocation  `json:"locations,omitempty"`
}

type SARIFMessageText struct {
	Text string `json:"text"`
}

type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
}

type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// RenderSARIF exports findings to SARIF format.
func RenderSARIF(w io.Writer, score *scorer.QualityScore) error {
	rulesMap := make(map[string]SARIFRule)
	var results []SARIFResult

	for _, f := range score.Findings {
		if _, exists := rulesMap[f.RuleID]; !exists {
			rulesMap[f.RuleID] = SARIFRule{
				ID:               f.RuleID,
				Name:             f.RuleName,
				ShortDescription: SARIFMessageText{Text: f.RuleName},
			}
		}

		level := "warning"
		switch f.Severity {
		case linter.SeverityError:
			level = "error"
		case linter.SeverityWarning:
			level = "warning"
		case linter.SeverityInfo:
			level = "note"
		}

		msg := fmt.Sprintf("[%s] %s Recommendation: %s", f.TargetName, f.Message, f.Recommendation)
		results = append(results, SARIFResult{
			RuleID:  f.RuleID,
			Level:   level,
			Message: SARIFMessageText{Text: msg},
			Locations: []SARIFLocation{
				{
					PhysicalLocation: SARIFPhysicalLocation{
						ArtifactLocation: SARIFArtifactLocation{
							URI: fmt.Sprintf("mcp://%s/%s", f.TargetType, f.TargetName),
						},
					},
				},
			},
		})
	}

	var rulesList []SARIFRule
	for _, r := range rulesMap {
		rulesList = append(rulesList, r)
	}

	report := SARIFReport{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:           "mcpq",
						Version:        "0.1.0",
						InformationURI: "https://github.com/akoserwal/mcpq",
						Rules:          rulesList,
					},
				},
				Results: results,
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
