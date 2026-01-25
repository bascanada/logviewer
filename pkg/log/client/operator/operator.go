// Package operator defines the supported filter operators.
package operator

const (
	// Equals checks for exact string equality.
	Equals   = "equals"
	Match    = "match"
	Wildcard = "wildcard"
	Exists   = "exists"
	Regex    = "regex"

	// Gt is the greater than operator (>).
	Gt  = "gt"
	// Gte is the greater than or equal operator (>=).
	Gte = "gte"
	// Lt is the less than operator (<).
	Lt  = "lt"
	// Lte is the less than or equal operator (<=).
	Lte = "lte"
)
