// Package ty provides utility types and constants.
package ty

import "time"

// Format is the standard timestamp format used.
const Format = time.RFC3339

// RegexTimestampFormat is the regex string to match timestamps.
const RegexTimestampFormat string = `(([0-9]*)-([0-9]*)-([0-9]*)T([0-9]*):([0-9]*):([0-9]*).([0-9]*)Z)`
