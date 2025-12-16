package client

import "github.com/bascanada/logviewer/pkg/ty"

// VariableDefinition describes a dynamic parameter for a search context.
// This provides metadata to UIs and LLMs about what inputs are expected.
type VariableDefinition struct {
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required,omitempty"`
}

type SearchRange struct {
	Lte  ty.Opt[string] `json:"lte" yaml:"lte"`
	Gte  ty.Opt[string] `json:"gte" yaml:"gte"`
	Last ty.Opt[string] `json:"last" yaml:"last"`
}

type RefreshOptions struct {
	Duration ty.Opt[string] `json:"duration,omitempty" yaml:"duration,omitempty"`
}

type FieldExtraction struct {
	GroupRegex     ty.Opt[string] `json:"groupRegex,omitempty" yaml:"groupRegex,omitempty"`
	KvRegex        ty.Opt[string] `json:"kvRegex,omitempty" yaml:"kvRegex,omitempty"`
	TimestampRegex ty.Opt[string] `json:"timestampRegex,omitempty" yaml:"timestampRegex,omitempty"`

	Json             ty.Opt[bool]   `json:"json,omitempty" yaml:"json,omitempty"`
	JsonMessageKey   ty.Opt[string] `json:"jsonMessageKey,omitempty" yaml:"jsonMessageKey,omitempty"`
	JsonLevelKey     ty.Opt[string] `json:"jsonLevelKey,omitempty" yaml:"jsonLevelKey,omitempty"`
	JsonTimestampKey ty.Opt[string] `json:"jsonTimestampKey,omitempty" yaml:"jsonTimestampKey,omitempty"`
}

type PrinterOptions struct {
	Template     ty.Opt[string] `json:"template,omitempty" yaml:"template,omitempty"`
	MessageRegex ty.Opt[string] `json:"messageRegex,omitempty" yaml:"messageRegex,omitempty"`
}

type LogSearch struct {
	// Current filterring fields (legacy - use Filter for complex queries)
	Fields ty.MS `json:"fields,omitempty" yaml:"fields,omitempty"`
	// Extra rules for filtering fields (legacy - use Filter for complex queries)
	FieldsCondition ty.MS `json:"fieldsCondition,omitempty" yaml:"fieldsCondition,omitempty"`

	// Filter is the new AST-based filter supporting nested logic (AND/OR/NOT)
	Filter *Filter `json:"filter,omitempty" yaml:"filter,omitempty"`

	// Range of the log query to do , depends of the system for full availability
	Range SearchRange `json:"range,omitempty" yaml:"range,omitempty"`

	// Max size of the request
	Size ty.Opt[int] `json:"size,omitempty" yaml:"size,omitempty"`

	// Refresh options for live data
	Refresh RefreshOptions `json:"refresh,omitempty" yaml:"refresh,omitempty"`

	// Options to configure the implementation with specific configuration for the search
	Options ty.MI `json:"options,omitempty" yaml:"options,omitempty"`

	// Token for fetching the next page of results
	PageToken ty.Opt[string] `json:"pageToken,omitempty" yaml:"pageToken,omitempty"`

	// Extra fields for field extraction for system without fieldging of log entry
	FieldExtraction FieldExtraction `json:"fieldExtraction,omitempty" yaml:"fieldExtraction,omitempty"`

	PrinterOptions PrinterOptions `json:"printerOptions,omitempty" yaml:"printerOptions,omitempty"`

	// Variables defines the dynamic inputs for this search context.
	// The map key is the variable name (e.g., "sessionId").
	Variables map[string]VariableDefinition `json:"variables,omitempty"`

	// Follow indicates if the search should continuously follow logs.
	Follow bool `json:"follow,omitempty" yaml:"follow,omitempty"`
}

// GetEffectiveFilter returns a unified filter tree that combines legacy Fields/FieldsCondition
// with the new Filter field. This allows backward compatibility while supporting new AST filters.
func (s *LogSearch) GetEffectiveFilter() *Filter {
	var allFilters []Filter

	// 1. Convert Legacy Fields to Filter Nodes
	for field, value := range s.Fields {
		op := "equals"
		if condition, ok := s.FieldsCondition[field]; ok && condition != "" {
			op = condition
		}

		allFilters = append(allFilters, Filter{
			Field: field,
			Op:    op,
			Value: value,
		})
	}

	// 2. Add the Explicit New Filter (if it exists)
	if s.Filter != nil {
		allFilters = append(allFilters, *s.Filter)
	}

	if len(allFilters) == 0 {
		return nil
	}

	// If there is only one condition, return it directly
	if len(allFilters) == 1 {
		return &allFilters[0]
	}

	// Otherwise, wrap everything in an implicit root "AND"
	return &Filter{
		Logic:   LogicAnd,
		Filters: allFilters,
	}
}

func (lr *LogSearch) MergeInto(logSeach *LogSearch) error {

	if lr.Fields == nil {
		lr.Fields = ty.MS{}
	}
	if lr.FieldsCondition == nil {
		lr.FieldsCondition = ty.MS{}
	}
	if lr.Options == nil {
		lr.Options = ty.MI{}
	}
	if lr.Variables == nil {
		lr.Variables = make(map[string]VariableDefinition)
	}

	for k, v := range logSeach.Variables {
		lr.Variables[k] = v
	}

	lr.Fields = ty.MergeM(lr.Fields, logSeach.Fields)
	lr.FieldsCondition = ty.MergeM(lr.FieldsCondition, logSeach.FieldsCondition)
	lr.Options = ty.MergeM(lr.Options, logSeach.Options)

	// Merge Filter: AND them together if both exist
	if logSeach.Filter != nil {
		if lr.Filter != nil {
			lr.Filter = &Filter{
				Logic:   LogicAnd,
				Filters: []Filter{*lr.Filter, *logSeach.Filter},
			}
		} else {
			lr.Filter = logSeach.Filter
		}
	}

	lr.Size.Merge(&logSeach.Size)
	lr.Refresh.Duration.Merge(&logSeach.Refresh.Duration)
	lr.FieldExtraction.GroupRegex.Merge(&logSeach.FieldExtraction.GroupRegex)
	lr.FieldExtraction.KvRegex.Merge(&logSeach.FieldExtraction.KvRegex)
	lr.FieldExtraction.TimestampRegex.Merge(&logSeach.FieldExtraction.TimestampRegex)
	lr.FieldExtraction.Json.Merge(&logSeach.FieldExtraction.Json)
	lr.FieldExtraction.JsonMessageKey.Merge(&logSeach.FieldExtraction.JsonMessageKey)
	lr.FieldExtraction.JsonLevelKey.Merge(&logSeach.FieldExtraction.JsonLevelKey)
	lr.FieldExtraction.JsonTimestampKey.Merge(&logSeach.FieldExtraction.JsonTimestampKey)
	lr.PrinterOptions.Template.Merge(&logSeach.PrinterOptions.Template)
	lr.PrinterOptions.MessageRegex.Merge(&logSeach.PrinterOptions.MessageRegex)
	lr.Range.Gte.Merge(&logSeach.Range.Gte)

	lr.Range.Lte.Merge(&logSeach.Range.Lte)
	lr.Range.Last.Merge(&logSeach.Range.Last)
	lr.PageToken.Merge(&logSeach.PageToken)

	return nil
}
