package suggest

// PatternType represents a progressive disclosure refactoring pattern.
type PatternType string

const (
	PatternPagination    PatternType = "CURSOR_PAGINATION"
	PatternFieldSelect   PatternType = "FIELD_PROJECTION"
	PatternSummaryDetail PatternType = "SUMMARY_DETAIL_SPLIT"
	PatternDecompose     PatternType = "MONOLITHIC_DECOMPOSITION"
	PatternEnumResource  PatternType = "ENUM_TO_RESOURCE"
)

// Suggestion represents an actionable refactoring recommendation for a tool.
type Suggestion struct {
	ToolName        string      `json:"toolName"`
	Pattern         PatternType `json:"pattern"`
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	EstimatedSaving string      `json:"estimatedSaving"`
	SchemaDiff      string      `json:"schemaDiff,omitempty"`
	TypeScriptCode  string      `json:"typeScriptCode,omitempty"`
	GoCode          string      `json:"goCode,omitempty"`
}
