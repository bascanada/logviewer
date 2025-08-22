package logclient

import (
	"fmt"
	"strings"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/client/operator"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

func escapeSplunkValue(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func getSearchRequest(logSearch *client.LogSearch) (ty.MS, error) {
	ms := ty.MS{
		"earliest_time": logSearch.Range.Gte.Value,
		"latest_time":   logSearch.Range.Lte.Value,
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
