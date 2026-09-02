package linter

import (
	"sync"
)

// Engine runs static and dynamic lint rules against MCP definitions and traces.
type Engine struct {
	mu           sync.RWMutex
	staticRules  []StaticRule
	dynamicRules []DynamicRule
}

// NewEngine creates a new linter Engine pre-registered with standard rules MCP001-MCP010.
func NewEngine() *Engine {
	e := &Engine{}
	e.RegisterStandardRules()
	return e
}

// RegisterRule registers a custom or standard rule.
func (e *Engine) RegisterRule(rule Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if sr, ok := rule.(StaticRule); ok {
		e.staticRules = append(e.staticRules, sr)
	}
	if dr, ok := rule.(DynamicRule); ok {
		e.dynamicRules = append(e.dynamicRules, dr)
	}
}

// RegisterStandardRules registers all built-in rules MCP001-MCP010.
func (e *Engine) RegisterStandardRules() {
	e.RegisterRule(&RuleMCP001Description{})
	e.RegisterRule(&RuleMCP002Monolithic{})
	e.RegisterRule(&RuleMCP003Unbounded{})
	e.RegisterRule(&RuleMCP004FieldSelect{})
	e.RegisterRule(&RuleMCP005Metadata{})
	e.RegisterRule(&RuleMCP006JSONKeys{})
	e.RegisterRule(&RuleMCP007EnumBloat{})
	e.RegisterRule(&RuleMCP008Duplicate{})
	e.RegisterRule(&RuleMCP009Docstring{})
	e.RegisterRule(&RuleMCP010PayloadSize{})
}

// LintStatic runs all static rules against the provided static MCP context.
func (e *Engine) LintStatic(ctx *StaticContext) []Finding {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var findings []Finding
	for _, r := range e.staticRules {
		res := r.CheckStatic(ctx)
		if len(res) > 0 {
			findings = append(findings, res...)
		}
	}
	return findings
}

// LintDynamic runs all dynamic rules against recorded traces.
func (e *Engine) LintDynamic(ctx *DynamicContext) []Finding {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var findings []Finding
	for _, r := range e.dynamicRules {
		res := r.CheckDynamic(ctx)
		if len(res) > 0 {
			findings = append(findings, res...)
		}
	}
	return findings
}

// LintAll runs both static and dynamic rules.
func (e *Engine) LintAll(sCtx *StaticContext, dCtx *DynamicContext) []Finding {
	var findings []Finding
	if sCtx != nil {
		findings = append(findings, e.LintStatic(sCtx)...)
	}
	if dCtx != nil {
		findings = append(findings, e.LintDynamic(dCtx)...)
	}
	return findings
}

// FilterFindings filters findings by minimum severity or rule ID.
func FilterFindings(findings []Finding, minSeverity Severity) []Finding {
	if minSeverity == "" {
		return findings
	}

	severityWeight := map[Severity]int{
		SeverityInfo:    1,
		SeverityWarning: 2,
		SeverityError:   3,
	}

	minWeight := severityWeight[minSeverity]
	var filtered []Finding
	for _, f := range findings {
		if severityWeight[f.Severity] >= minWeight {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
