package logclient

import (
	"fmt"
	"strings"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/operator"
	"github.com/bascanada/logviewer/pkg/ty"
)

func escapeSplunkValue(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
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
	var regexQuery strings.Builder

	if index, ok := logSearch.Options.GetStringOk("index"); ok {
		query.WriteString(fmt.Sprintf("index=%s", index))
	}

	for k, v := range logSearch.Fields {
		op := logSearch.FieldsCondition[k]
		if op == "" {
			op = operator.Equals
		}

		// Support free-text search items: if the field key is empty or a
		// sentinel value of "_" then treat the value as a plain search
		// token (or quoted phrase) instead of a key=value expression.
		if k == "" || k == "_" {
			if query.Len() > 0 {
				query.WriteString(" ")
			}

			// For regex operator on free-text, convert to a regex against
			// the raw event text.
			if op == operator.Regex {
				query.WriteString(fmt.Sprintf(`regex _raw="%s"`, escapeSplunkValue(v)))
				continue
			}

			// Quote the phrase if it contains spaces.
			if strings.Contains(v, " ") {
				query.WriteString(fmt.Sprintf(`"%s"`, escapeSplunkValue(v)))
			} else {
				query.WriteString(escapeSplunkValue(v))
			}

			continue
		}

		if op == operator.Regex {
			if regexQuery.Len() > 0 {
				regexQuery.WriteString(" | ")
			}
			regexQuery.WriteString(fmt.Sprintf(`regex %s="%s"`, k, escapeSplunkValue(v)))
			continue
		}

		if query.Len() > 0 {
			query.WriteString(" ")
		}

		switch op {
		case operator.Equals, operator.Match:
			query.WriteString(fmt.Sprintf(`%s="%s"`, k, escapeSplunkValue(v)))
		case operator.Wildcard:
			query.WriteString(fmt.Sprintf(`%s="%s*"`, k, escapeSplunkValue(v)))
		case operator.Exists:
			query.WriteString(fmt.Sprintf(`%s=*`, k))
		}
	}

	if regexQuery.Len() > 0 {
		if query.Len() > 0 {
			query.WriteString(" ")
		}
		query.WriteString("| ")
		query.WriteString(regexQuery.String())
	}

	ms["search"] = query.String()

	return ms, nil
}
