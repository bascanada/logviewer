package client

import "github.com/bascanada/logviewer/pkg/ty"

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

	// Token for fetching the next page of results
	PageToken ty.Opt[string] `json:"pageToken,omitempty"`

	// Extra fields for field extraction for system without fieldging of log entry
	FieldExtraction FieldExtraction `json:"fieldExtraction,omitempty"`

	PrinterOptions PrinterOptions `json:"printerOptions,omitempty"`
}

func (lr *LogSearch) MergeInto(logSeach *LogSearch) error {

	if lr.Fields == nil {
		lr.Fields = ty.MS{}
	}
	if lr.Fields == nil {
		lr.Fields = ty.MS{}
	}
	if lr.Options == nil {
		lr.Options = ty.MI{}
	}

	lr.Fields = ty.MergeM(lr.Fields, logSeach.Fields)
	lr.Fields = ty.MergeM(lr.Fields, logSeach.Fields)
	lr.Options = ty.MergeM(lr.Options, logSeach.Options)

	lr.Size.Merge(&logSeach.Size)
	lr.Refresh.Duration.Merge(&logSeach.Refresh.Duration)
	lr.FieldExtraction.Regex.Merge(&logSeach.FieldExtraction.Regex)
	lr.PrinterOptions.Template.Merge(&logSeach.PrinterOptions.Template)
	lr.Range.Gte.Merge(&logSeach.Range.Gte)
	lr.Range.Lte.Merge(&logSeach.Range.Lte)
	lr.Range.Last.Merge(&logSeach.Range.Last)
	lr.PageToken.Merge(&logSeach.PageToken)

	return nil
}
