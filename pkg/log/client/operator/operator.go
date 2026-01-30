// Package operator defines the supported filter operators.
package operator

const (
	// Equals checks for exact string equality.
	Equals = "equals"
	// Match performs a match query.
	Match = "match"
	// Wildcard performs a wildcard query.
	Wildcard = "wildcard"
	// Exists checks if a field exists.
	Exists = "exists"
	// Regex performs a regular expression match.
	Regex = "regex"

	// Gt is the greater than operator (>).
	Gt  = "gt"
	// Gte is the greater than or equal operator (>=).
	Gte = "gte"
	// Lt is the less than operator (<).
	Lt  = "lt"
	// Lte is the less than or equal operator (<=).
	Lte = "lte"
)
