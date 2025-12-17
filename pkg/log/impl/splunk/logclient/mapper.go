package logclient

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/operator"
	"github.com/bascanada/logviewer/pkg/ty"
)

// transformingCommands lists Splunk commands that transform events into results.
// These commands require fetching from /results endpoint instead of /events.
var transformingCommands = []string{
	"stats", "chart", "timechart", "top", "rare",
	"transaction", "cluster", "kmeans",
	"eventstats", "streamstats",
	"bucket", "bin",
	"predict", "trendline",
	"geostats", "sichart", "sitimechart",
	"mstats", "tstats",
	"table", "fields",
}

// transformingCommandPattern matches pipe followed by a transforming command.
// Uses word boundary to avoid matching partial words (e.g., "topaz" shouldn't match "top").
var transformingCommandPattern *regexp.Regexp

func init() {
	// Build pattern: | followed by optional whitespace, then one of the commands as a word
	pattern := `\|\s*(` + strings.Join(transformingCommands, "|") + `)(?:\s|$)`
	transformingCommandPattern = regexp.MustCompile("(?i)" + pattern)
}

// ContainsTransformingCommand checks if a Splunk query contains transforming commands
// that require results to be fetched from the /results endpoint instead of /events.
func ContainsTransformingCommand(query string) bool {
	return transformingCommandPattern.MatchString(query)
}

func escapeSplunkValue(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

// buildSplunkCondition builds a single condition for Splunk search.
// Returns the condition string and a boolean indicating if it's a regex (needs pipe).
func buildSplunkCondition(f *client.Filter) (condition string, isRegex bool) {
	if f.Field == "" {
		return "", false
	}

	op := f.Op
	if op == "" {
		op = operator.Equals
	}

	// Handle free-text search (field is "_" or empty)
	if f.Field == "_" {
		if op == operator.Regex {
			return fmt.Sprintf(`regex _raw="%s"`, escapeSplunkValue(f.Value)), true
		}
		if strings.Contains(f.Value, " ") {
			return fmt.Sprintf(`"%s"`, escapeSplunkValue(f.Value)), false
		}
		return escapeSplunkValue(f.Value), false
	}

	switch op {
	case operator.Regex:
		return fmt.Sprintf(`regex %s="%s"`, f.Field, escapeSplunkValue(f.Value)), true
	case operator.Wildcard:
		return fmt.Sprintf(`%s="%s*"`, f.Field, escapeSplunkValue(f.Value)), false
	case operator.Exists:
		return fmt.Sprintf(`%s=*`, f.Field), false
	default: // equals, match
		return fmt.Sprintf(`%s="%s"`, f.Field, escapeSplunkValue(f.Value)), false
	}
}

// buildSplunkQuery recursively builds a Splunk search query from a Filter AST.
// It returns the main query string and a slice of regex conditions that need pipe commands.
func buildSplunkQuery(f *client.Filter) (query string, regexConditions []string) {
	if f == nil {
		return "", nil
	}

	// Handle Leaf (Condition)
	if f.Field != "" {
		cond, isRegex := buildSplunkCondition(f)
		if isRegex {
			return "", []string{cond}
		}
		return cond, nil
	}

	// Handle Branch (Group)
	if f.Logic == "" || len(f.Filters) == 0 {
		return "", nil
	}

	var parts []string
	var allRegex []string

	for _, child := range f.Filters {
		childQuery, childRegex := buildSplunkQuery(&child)
		if childQuery != "" {
			parts = append(parts, childQuery)
		}
		allRegex = append(allRegex, childRegex...)
	}

	if len(parts) == 0 {
		return "", allRegex
	}

	var result string
	switch f.Logic {
	case client.LogicAnd:
		// Splunk uses space for implicit AND
		result = strings.Join(parts, " ")
	case client.LogicOr:
		// Splunk uses OR keyword
		result = strings.Join(parts, " OR ")
	case client.LogicNot:
		// NOT applies to all children (ANDed together, then inverted)
		inner := strings.Join(parts, " ")
		result = fmt.Sprintf("NOT (%s)", inner)
	}

	// Wrap in parentheses if multiple parts (for proper precedence)
	if len(parts) > 1 && f.Logic != client.LogicNot {
		result = "(" + result + ")"
	}

	return result, allRegex
}

func getSearchRequest(logSearch *client.LogSearch) (ty.MS, error) {
	ms := ty.MS{
		"earliest_time": logSearch.Range.Gte.Value,
		"latest_time":   logSearch.Range.Lte.Value,
	}

	// If the caller provided a `last` duration (e.g. "1min"), prefer that
	// over explicit gte/lte. We translate it to an earliest_time of
	// "-<last>" and latest_time of "now" which Splunk understands as a
	// relative time window.
	if logSearch.Range.Last.Value != "" {
		ms["earliest_time"] = "-" + logSearch.Range.Last.Value
		ms["latest_time"] = "now"
	}

	var query strings.Builder
	hasNativeQuery := logSearch.NativeQuery.Set && logSearch.NativeQuery.Value != ""

	// 1. Start with Native Query if provided
	if hasNativeQuery {
		query.WriteString(logSearch.NativeQuery.Value)
	}

	// 2. Add index if specified (append as pipe filter if native query exists)
	if index, ok := logSearch.Options.GetStringOk("index"); ok {
		if hasNativeQuery {
			query.WriteString(fmt.Sprintf(" | search index=%s", index))
		} else {
			query.WriteString(fmt.Sprintf("index=%s", index))
		}
	}

	// 3. Get the effective filter (combines legacy Fields with new Filter)
	effectiveFilter := logSearch.GetEffectiveFilter()

	if effectiveFilter != nil {
		filterQuery, regexConditions := buildSplunkQuery(effectiveFilter)

		if filterQuery != "" {
			if query.Len() > 0 {
				// Use pipe search if we have a native query, otherwise space join
				if hasNativeQuery {
					query.WriteString(" | search ")
				} else {
					query.WriteString(" ")
				}
			}
			query.WriteString(filterQuery)
		}

		// Add regex conditions as pipe commands
		for _, regex := range regexConditions {
			if query.Len() > 0 {
				query.WriteString(" | ")
			}
			query.WriteString(regex)
		}
	}

	// Add fields selection if specified
	if fields, ok := logSearch.Options.GetListOfStringsOk("fields"); ok {
		if len(fields) > 0 {
			query.WriteString(fmt.Sprintf(" | fields + %s", strings.Join(fields, ", ")))
		}
	}

	ms["search"] = query.String()

	return ms, nil
}
