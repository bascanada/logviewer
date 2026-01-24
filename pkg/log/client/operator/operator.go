// Package operator defines the supported filter operators.
package operator

const (
	Equals   = "equals"
	Match    = "match"
	Wildcard = "wildcard"
	Exists   = "exists"
	Regex    = "regex"
	// Comparison operators for hl-compatible syntax
	Gt  = "gt"  // >
	Gte = "gte" // >=
	Lt  = "lt"  // <
	Lte = "lte" // <=
)
