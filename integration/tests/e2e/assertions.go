//go:build integration

package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var verboseAssertions = os.Getenv("VERBOSE_ASSERTIONS") == "1" || os.Getenv("VERBOSE_ASSERTIONS") == "true"

type LogResult struct {
	t       *testing.T
	logs    []map[string]interface{}
	verbose bool
}

func Expect(t *testing.T, logs []map[string]interface{}) *LogResult {
	t.Helper()
	return &LogResult{t: t, logs: logs, verbose: verboseAssertions}
}

func (r *LogResult) Verbose() *LogResult {
	r.verbose = true
	return r
}

func (r *LogResult) Silent() *LogResult {
	r.verbose = false
	return r
}

func (r *LogResult) Count(expected int) *LogResult {
	r.t.Helper()
	actual := len(r.logs)
	if r.verbose {
		r.t.Logf("✓ Asserting count: expected=%d, actual=%d", expected, actual)
	}
	assert.Len(r.t, r.logs, expected, "Unexpected log count")
	return r
}

func (r *LogResult) AtLeast(min int) *LogResult {
	r.t.Helper()
	actual := len(r.logs)
	if r.verbose {
		r.t.Logf("✓ Asserting at least: min=%d, actual=%d", min, actual)
	}
	assert.GreaterOrEqual(r.t, len(r.logs), min, "Too few logs returned")
	return r
}

func (r *LogResult) AtMost(max int) *LogResult {
	r.t.Helper()
	actual := len(r.logs)
	if r.verbose {
		r.t.Logf("✓ Asserting at most: max=%d, actual=%d", max, actual)
	}
	assert.LessOrEqual(r.t, len(r.logs), max, "Too many logs returned")
	return r
}

func (r *LogResult) Between(min, max int) *LogResult {
	r.t.Helper()
	count := len(r.logs)
	if r.verbose {
		r.t.Logf("✓ Asserting between: min=%d, max=%d, actual=%d", min, max, count)
	}
	assert.GreaterOrEqual(r.t, count, min, "Too few logs returned")
	assert.LessOrEqual(r.t, count, max, "Too many logs returned")
	return r
}

func (r *LogResult) IsEmpty() *LogResult {
	r.t.Helper()
	if r.verbose {
		r.t.Logf("✓ Asserting is empty: actual count=%d", len(r.logs))
	}
	assert.Empty(r.t, r.logs, "Expected no logs, but got some")
	return r
}

func (r *LogResult) IsNotEmpty() *LogResult {
	r.t.Helper()
	if r.verbose {
		r.t.Logf("✓ Asserting is not empty: actual count=%d", len(r.logs))
	}
	assert.NotEmpty(r.t, r.logs, "Expected logs, but got none")
	return r
}

func (r *LogResult) All(checks ...LogCheck) *LogResult {
	r.t.Helper()
	if r.verbose {
		checkNames := make([]string, len(checks))
		for i, check := range checks {
			checkNames[i] = check.Description()
		}
		r.t.Logf("✓ Asserting all %d logs match checks: %s", len(r.logs), strings.Join(checkNames, ", "))
	}
	passCount := 0
	for i, log := range r.logs {
		allPassed := true
		for _, check := range checks {
			if !check.Validate(log) {
				allPassed = false
				assert.Fail(r.t, fmt.Sprintf("Log #%d failed check: %s", i, check.Description()),
					"Log content: %v", log)
			}
		}
		if allPassed {
			passCount++
		}
	}
	if r.verbose && passCount == len(r.logs) {
		r.t.Logf("  ✓ All %d logs passed all checks", passCount)
	}
	return r
}

func (r *LogResult) Any(checks ...LogCheck) *LogResult {
	r.t.Helper()
	if r.verbose {
		checkNames := make([]string, len(checks))
		for i, check := range checks {
			checkNames[i] = check.Description()
		}
		r.t.Logf("✓ Asserting at least one log matches all checks: %s", strings.Join(checkNames, ", "))
	}
	for _, log := range r.logs {
		passedAll := true
		for _, check := range checks {
			if !check.Validate(log) {
				passedAll = false
				break
			}
		}
		if passedAll {
			if r.verbose {
				r.t.Logf("  ✓ Found matching log: %v", log)
			}
			return r
		}
	}
	checkDescriptions := make([]string, len(checks))
	for i, check := range checks {
		checkDescriptions[i] = check.Description()
	}
	assert.Fail(r.t, "No log matched all the provided criteria",
		"Checks: %s", strings.Join(checkDescriptions, ", "))
	return r
}

func (r *LogResult) First(checks ...LogCheck) *LogResult {
	r.t.Helper()
	if len(r.logs) == 0 {
		assert.Fail(r.t, "Cannot assert on First log: result set is empty")
		return r
	}
	if r.verbose {
		checkNames := make([]string, len(checks))
		for i, check := range checks {
			checkNames[i] = check.Description()
		}
		r.t.Logf("✓ Asserting first log matches checks: %s", strings.Join(checkNames, ", "))
	}
	allPassed := true
	for _, check := range checks {
		if !check.Validate(r.logs[0]) {
			allPassed = false
			assert.Fail(r.t, fmt.Sprintf("First log failed check: %s", check.Description()),
				"Log content: %v", r.logs[0])
		}
	}
	if r.verbose && allPassed {
		r.t.Logf("  ✓ First log passed all checks")
	}
	return r
}

func (r *LogResult) None(checks ...LogCheck) *LogResult {
	r.t.Helper()
	if r.verbose {
		checkNames := make([]string, len(checks))
		for i, check := range checks {
			checkNames[i] = check.Description()
		}
		r.t.Logf("✓ Asserting no logs match all checks: %s", strings.Join(checkNames, ", "))
	}
	matchCount := 0
	for i, log := range r.logs {
		passedAll := true
		for _, check := range checks {
			if !check.Validate(log) {
				passedAll = false
				break
			}
		}
		if passedAll {
			matchCount++
			checkDescriptions := make([]string, len(checks))
			for j, check := range checks {
				checkDescriptions[j] = check.Description()
			}
			assert.Fail(r.t, fmt.Sprintf("Log #%d unexpectedly matched all criteria: %s",
				i, strings.Join(checkDescriptions, ", ")),
				"Log content: %v", log)
			return r
		}
	}
	if r.verbose {
		r.t.Logf("  ✓ Confirmed: no logs matched (checked %d logs)", len(r.logs))
	}
	return r
}

type LogCheck interface {
	Validate(log map[string]interface{}) bool
	Description() string
}

type simpleCheck struct {
	validator func(map[string]interface{}) bool
	desc      string
}

func (c simpleCheck) Validate(log map[string]interface{}) bool {
	return c.validator(log)
}

func (c simpleCheck) Description() string {
	return c.desc
}

func FieldEquals(field string, expected interface{}) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			val, ok := log[field]
			return ok && assert.ObjectsAreEqual(expected, val)
		},
		desc: fmt.Sprintf("field '%s' equals '%v'", field, expected),
	}
}

func FieldContains(field string, substring string) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			val, ok := log[field]
			if !ok {
				return false
			}
			strVal, isStr := val.(string)
			return isStr && strings.Contains(strVal, substring)
		},
		desc: fmt.Sprintf("field '%s' contains '%s'", field, substring),
	}
}

func FieldMatches(field string, pattern string) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			val, ok := log[field]
			if !ok {
				return false
			}
			strVal, isStr := val.(string)
			if !isStr {
				return false
			}
			if strings.Contains(pattern, "*") {
				parts := strings.Split(pattern, "*")
				pos := 0
				for _, part := range parts {
					if part == "" {
						continue
					}
					idx := strings.Index(strVal[pos:], part)
					if idx == -1 {
						return false
					}
					pos += idx + len(part)
				}
				return true
			}
			return strVal == pattern
		},
		desc: fmt.Sprintf("field '%s' matches pattern '%s'", field, pattern),
	}
}

func FieldNotPresent(field string) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			_, ok := log[field]
			return !ok
		},
		desc: fmt.Sprintf("field '%s' not present", field),
	}
}

func FieldPresent(field string) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			_, ok := log[field]
			return ok
		},
		desc: fmt.Sprintf("field '%s' is present", field),
	}
}

func FieldNotEmpty(field string) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			val, ok := log[field]
			if !ok {
				return false
			}
			strVal, isStr := val.(string)
			return isStr && strVal != ""
		},
		desc: fmt.Sprintf("field '%s' is not empty", field),
	}
}

func FieldOneOf(field string, allowedValues ...interface{}) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			val, ok := log[field]
			if !ok {
				return false
			}
			for _, allowed := range allowedValues {
				if assert.ObjectsAreEqual(val, allowed) {
					return true
				}
			}
			return false
		},
		desc: fmt.Sprintf("field '%s' is one of %v", field, allowedValues),
	}
}

func DateBetween(field string, start, end time.Time) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			val, ok := log[field]
			if !ok {
				return false
			}
			strVal, isStr := val.(string)
			if !isStr {
				return false
			}
			formats := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02T15:04:05.999Z",
				"2006-01-02T15:04:05Z",
				"2006-01-02 15:04:05",
			}
			var t time.Time
			var err error
			for _, format := range formats {
				t, err = time.Parse(format, strVal)
				if err == nil {
					break
				}
			}
			if err != nil {
				return false
			}
			return (t.After(start) || t.Equal(start)) && (t.Before(end) || t.Equal(end))
		},
		desc: fmt.Sprintf("field '%s' between %s and %s", field, start.Format(time.RFC3339), end.Format(time.RFC3339)),
	}
}

func DateAfter(field string, after time.Time) LogCheck {
	return simpleCheck{
		validator: func(log map[string]interface{}) bool {
			val, ok := log[field]
			if !ok {
				return false
			}
			strVal, isStr := val.(string)
			if !isStr {
				return false
			}
			formats := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05.999Z", "2006-01-02T15:04:05Z"}
			for _, format := range formats {
				if t, err := time.Parse(format, strVal); err == nil {
					return t.After(after)
				}
			}
			return false
		},
		desc: fmt.Sprintf("field '%s' after %s", field, after.Format(time.RFC3339)),
	}
}

func HasTraceID() LogCheck {
	return FieldNotEmpty("trace_id")
}

func IsErrorLevel() LogCheck {
	return FieldEquals("level", "ERROR")
}

func IsFromApp(appName string) LogCheck {
	return FieldEquals("app", appName)
}

func IsFromContext(contextID string) LogCheck {
	return FieldEquals("context_id", contextID)
}

type ValuesResult struct {
	t       *testing.T
	values  map[string][]string
	verbose bool
}

func ExpectValues(t *testing.T, values map[string][]string) *ValuesResult {
	t.Helper()
	return &ValuesResult{
		t:       t,
		values:  values,
		verbose: verboseAssertions,
	}
}

func (v *ValuesResult) Verbose() *ValuesResult {
	v.verbose = true
	return v
}

func (v *ValuesResult) Silent() *ValuesResult {
	v.verbose = false
	return v
}

func (v *ValuesResult) FieldHasValues(field string, expected ...string) *ValuesResult {
	v.t.Helper()
	if v.verbose {
		v.t.Logf("✓ Asserting field '%s' has values: %v", field, expected)
	}
	actual, ok := v.values[field]
	assert.True(v.t, ok, "Field '%s' not found in results", field)
	if v.verbose && ok {
		v.t.Logf("  Actual values for '%s': %v", field, actual)
	}
	for _, exp := range expected {
		assert.Contains(v.t, actual, exp, "Field '%s' missing expected value '%s'", field, exp)
	}
	if v.verbose {
		v.t.Logf("  ✓ All expected values found")
	}
	return v
}

func (v *ValuesResult) FieldHasExactValues(field string, expected ...string) *ValuesResult {
	v.t.Helper()
	if v.verbose {
		v.t.Logf("✓ Asserting field '%s' has exact values: %v", field, expected)
	}
	actual, ok := v.values[field]
	assert.True(v.t, ok, "Field '%s' not found in results", field)
	if v.verbose && ok {
		v.t.Logf("  Actual values for '%s': %v", field, actual)
	}
	assert.ElementsMatch(v.t, expected, actual, "Field '%s' has different values", field)
	if v.verbose {
		v.t.Logf("  ✓ Values match exactly")
	}
	return v
}

func (v *ValuesResult) FieldValueCount(field string, count int) *ValuesResult {
	v.t.Helper()
	actual, ok := v.values[field]
	actualCount := 0
	if ok {
		actualCount = len(actual)
	}
	if v.verbose {
		v.t.Logf("✓ Asserting field '%s' value count: expected=%d, actual=%d", field, count, actualCount)
	}
	assert.True(v.t, ok, "Field '%s' not found in results", field)
	assert.Len(v.t, actual, count, "Field '%s' has unexpected value count", field)
	return v
}

func (v *ValuesResult) HasField(field string) *ValuesResult {
	v.t.Helper()
	if v.verbose {
		v.t.Logf("✓ Asserting field '%s' is present", field)
	}
	_, ok := v.values[field]
	assert.True(v.t, ok, "Field '%s' not found in results", field)
	if v.verbose && ok {
		valueCount := len(v.values[field])
		v.t.Logf("  ✓ Field found with %d values", valueCount)
	}
	return v
}

func (v *ValuesResult) HasFields(fields ...string) *ValuesResult {
	v.t.Helper()
	if v.verbose {
		v.t.Logf("✓ Asserting multiple fields present: %v", fields)
	}
	for _, field := range fields {
		if v.verbose {
			v.verbose = false // Suppress sub-logging
			v.HasField(field)
			v.verbose = true
		} else {
			v.HasField(field)
		}
	}
	if v.verbose {
		v.t.Logf("  ✓ All %d fields found", len(fields))
	}
	return v
}
