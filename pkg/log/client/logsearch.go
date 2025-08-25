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
	Lte  ty.Opt[string] `json:"lte"`
	Gte  ty.Opt[string] `json:"gte"`
	Last ty.Opt[string] `json:"last"`
}

type RefreshOptions struct {
	Follow   ty.Opt[bool]   `json:"follow,omitempty"`
	Duration ty.Opt[string] `json:"duration,omitempty"`
}

type FieldExtraction struct {
	Regex          ty.Opt[string] `json:"regex,omitempty"`
	TimestampRegex ty.Opt[string] `json:"date,omitempty"`
}

type PrinterOptions struct {
	Template ty.Opt[string] `json:"template,omitempty"`
}

type LogSearch struct {
	// Current filterring fields
	Fields ty.MS `json:"fields,omitempty"`
	// Extra rules for filtering fields
	FieldsCondition ty.MS `json:"fieldsCondition,omitempty"`

	// Range of the log query to do , depends of the system for full availability
	Range SearchRange `json:"range,omitempty"`

	// Max size of the request
	Size ty.Opt[int] `json:"size,omitempty"`

	// Refresh options for live data
	Refresh RefreshOptions `json:"refresh,omitempty"`

	// Options to configure the implementation with specific configuration for the search
	Options ty.MI `json:"options,omitempty"`

	// Extra fields for field extraction for system without fieldging of log entry
	FieldExtraction FieldExtraction `json:"fieldExtraction,omitempty"`

	PrinterOptions PrinterOptions `json:"printerOptions,omitempty"`

	// Variables defines the dynamic inputs for this search context.
	// The map key is the variable name (e.g., "sessionId").
	Variables map[string]VariableDefinition `json:"variables,omitempty"`
}

func (lr *LogSearch) MergeInto(logSeach *LogSearch) error {
	if lr.Fields == nil {
		lr.Fields = ty.MS{}
	}
	if lr.Options == nil {
		lr.Options = ty.MI{}
	}

	lr.Fields = ty.MergeM(lr.Fields, logSeach.Fields)
	lr.Options = ty.MergeM(lr.Options, logSeach.Options)

	lr.Size.Merge(&logSeach.Size)
	lr.Refresh.Duration.Merge(&logSeach.Refresh.Duration)
	lr.FieldExtraction.Regex.Merge(&logSeach.FieldExtraction.Regex)
	lr.PrinterOptions.Template.Merge(&logSeach.PrinterOptions.Template)
	lr.Range.Gte.Merge(&logSeach.Range.Gte)
	lr.Range.Lte.Merge(&logSeach.Range.Lte)
	lr.Range.Last.Merge(&logSeach.Range.Last)

	if lr.Variables == nil {
		lr.Variables = make(map[string]VariableDefinition)
	}

	for k, v := range logSeach.Variables {
		lr.Variables[k] = v
	}

	return nil
}
