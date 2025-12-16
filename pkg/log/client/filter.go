package client

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bascanada/logviewer/pkg/log/client/operator"
)

type LogicOperator string

const (
	LogicAnd LogicOperator = "AND"
	LogicOr  LogicOperator = "OR"
	LogicNot LogicOperator = "NOT"
)

// Filter represents a recursive filter AST node.
// It can be either a leaf node (condition) or a branch node (group).
type Filter struct {
	// --- Leaf Node (Condition) ---
	// If Field is set, this is a condition
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
	Op    string `json:"op,omitempty" yaml:"op,omitempty"`    // e.g., "equals", "regex", "wildcard", "exists", "match"
	Value string `json:"value,omitempty" yaml:"value,omitempty"`

	// --- Branch Node (Group) ---
	// If Logic is set, this is a group
	Logic   LogicOperator `json:"logic,omitempty" yaml:"logic,omitempty"`
	Filters []Filter      `json:"filters,omitempty" yaml:"filters,omitempty"`
}

// Validate checks if the filter is structurally valid.
func (f *Filter) Validate() error {
	if f == nil {
		return nil
	}

	isLeaf := f.Field != ""
	isBranch := f.Logic != ""

	// A filter must be either a leaf or a branch, not both
	if isLeaf && isBranch {
		return fmt.Errorf("filter cannot have both 'field' and 'logic' set")
	}

	// Empty filter (neither leaf nor branch) is valid and means "match all"
	if !isLeaf && !isBranch {
		return nil
	}

	// Validate leaf node
	if isLeaf {
		// Validate operator
		switch f.Op {
		case "", operator.Equals, operator.Match, operator.Wildcard, operator.Exists, operator.Regex:
			// valid
		default:
			return fmt.Errorf("invalid operator: %s", f.Op)
		}

		// 'exists' operator doesn't need a value, others do
		if f.Op != operator.Exists && f.Value == "" {
			return fmt.Errorf("filter with field '%s' requires a value (unless op is 'exists')", f.Field)
		}

		// Leaf nodes shouldn't have children
		if len(f.Filters) > 0 {
			return fmt.Errorf("leaf filter (field='%s') cannot have nested filters", f.Field)
		}
	}

	// Validate branch node
	if isBranch {
		// Validate logic operator
		switch f.Logic {
		case LogicAnd, LogicOr, LogicNot:
			// valid
		default:
			return fmt.Errorf("invalid logic operator: %s", f.Logic)
		}

		// NOT should ideally have at least one child
		if f.Logic == LogicNot && len(f.Filters) == 0 {
			return fmt.Errorf("NOT filter must have at least one child filter")
		}

		// Branch nodes shouldn't have leaf properties
		if f.Value != "" {
			return fmt.Errorf("branch filter (logic='%s') should not have a value", f.Logic)
		}

		// Recursively validate children
		for i, child := range f.Filters {
			if err := child.Validate(); err != nil {
				return fmt.Errorf("filter[%d]: %w", i, err)
			}
		}
	}

	return nil
}

// Match evaluates the filter against a LogEntry (client-side filtering).
func (f *Filter) Match(entry LogEntry) bool {
	if f == nil {
		return true
	}

	// Handle Branch (Group)
	if f.Logic != "" {
		return f.matchBranch(entry)
	}

	// Handle Leaf (Condition)
	if f.Field != "" {
		return f.matchLeaf(entry)
	}

	// Empty filter matches everything
	return true
}

func (f *Filter) matchBranch(entry LogEntry) bool {
	if len(f.Filters) == 0 {
		return true // Empty group matches everything
	}

	switch f.Logic {
	case LogicAnd:
		for _, child := range f.Filters {
			if !child.Match(entry) {
				return false
			}
		}
		return true

	case LogicOr:
		for _, child := range f.Filters {
			if child.Match(entry) {
				return true
			}
		}
		return false

	case LogicNot:
		// NOT inverts the result of all children ANDed together
		for _, child := range f.Filters {
			if !child.Match(entry) {
				return true // If any child doesn't match, NOT matches
			}
		}
		return false
	}

	return true
}

func (f *Filter) matchLeaf(entry LogEntry) bool {
	// Handle special "_" sentinel for raw message search
	if f.Field == "_" {
		return f.matchValue(entry.Message)
	}

	// Use LogEntry.Field() for consistent field access (handles case-insensitivity and struct fields)
	fieldValRaw := entry.Field(f.Field)

	// Handle "exists" operator
	if f.Op == operator.Exists {
		return fieldValRaw != "" && fieldValRaw != nil
	}

	// Convert to string for comparison
	fieldVal := toString(fieldValRaw)

	// If field is missing/empty, no match (except for exists which is handled above)
	if fieldVal == "" {
		return false
	}

	return f.matchValue(fieldVal)
}

func (f *Filter) matchValue(fieldVal string) bool {
	switch f.Op {
	case operator.Regex:
		matched, err := regexp.MatchString(f.Value, fieldVal)
		if err != nil {
			return false
		}
		return matched

	case operator.Wildcard:
		// Convert glob pattern to regex: * -> .*, ? -> .
		pattern := regexp.QuoteMeta(f.Value)
		pattern = strings.ReplaceAll(pattern, `\*`, `.*`)
		pattern = strings.ReplaceAll(pattern, `\?`, `.`)
		pattern = "^" + pattern + "$"
		matched, err := regexp.MatchString(pattern, fieldVal)
		if err != nil {
			return false
		}
		return matched

	case operator.Match:
		// Match is a case-insensitive contains
		return strings.Contains(strings.ToLower(fieldVal), strings.ToLower(f.Value))

	case "", operator.Equals:
		return fieldVal == f.Value

	default:
		// Unknown operator, default to equals
		return fieldVal == f.Value
	}
}

// toString converts an interface{} to string for comparison
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}
