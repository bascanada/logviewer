package logclient

import (
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/operator"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

func TestSearchRequest(t *testing.T) {

	t.Run("simple search with index and one equals condition", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"application_name": "wq.services.pet"},
			FieldsCondition: ty.MS{},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Equal(t, `index=nonprod application_name="wq.services.pet"`, requestBodyFields["search"])
	})

	t.Run("free text search token", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"_": "error occurred"},
			FieldsCondition: ty.MS{"_": ""},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		// should include index and quoted phrase after it
		assert.Equal(t, `index=nonprod "error occurred"`, requestBodyFields["search"])
	})

	t.Run("search with multiple equals conditions", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"application_name": "wq.services.pet", "trace_id": "1234"},
			FieldsCondition: ty.MS{},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Contains(t, requestBodyFields["search"], `index=nonprod`)
		assert.Contains(t, requestBodyFields["search"], `application_name="wq.services.pet"`)
		assert.Contains(t, requestBodyFields["search"], `trace_id="1234"`)
	})

	t.Run("search with wildcard condition", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"application_name": "wq.services"},
			FieldsCondition: ty.MS{"application_name": operator.Wildcard},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Equal(t, `index=nonprod application_name="wq.services*"`, requestBodyFields["search"])
	})

	t.Run("search with wildcard condition and spaces", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"application_name": "wq services"},
			FieldsCondition: ty.MS{"application_name": operator.Wildcard},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Equal(t, `index=nonprod application_name="wq services*"`, requestBodyFields["search"])
	})

	t.Run("search with exists condition", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"trace_id": ""},
			FieldsCondition: ty.MS{"trace_id": operator.Exists},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Equal(t, `index=nonprod trace_id=*`, requestBodyFields["search"])
	})

	t.Run("search with regex condition", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"message": "(error|fail)"},
			FieldsCondition: ty.MS{"message": operator.Regex},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Equal(t, `index=nonprod | regex message="(error|fail)"`, requestBodyFields["search"])
	})

	t.Run("complex search with multiple operators", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields: ty.MS{
				"application_name": "wq.services.pet",
				"http_method":      "GET",
				"message":          "(error|fail)",
				"trace_id":         "",
			},
			FieldsCondition: ty.MS{
				"application_name": operator.Wildcard,
				"http_method":      operator.Equals,
				"message":          operator.Regex,
				"trace_id":         operator.Exists,
			},
			Options: ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Contains(t, requestBodyFields["search"], `index=nonprod`)
		assert.Contains(t, requestBodyFields["search"], `application_name="wq.services.pet*"`)
		assert.Contains(t, requestBodyFields["search"], `http_method="GET"`)
		assert.Contains(t, requestBodyFields["search"], `trace_id=*`)
		assert.Contains(t, requestBodyFields["search"], `| regex message="(error|fail)"`)
	})

	t.Run("search with value containing spaces", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"message": "this is a test"},
			FieldsCondition: ty.MS{},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Equal(t, `index=nonprod message="this is a test"`, requestBodyFields["search"])
	})

	t.Run("search with value containing double quotes", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"message": `this is a "test"`},
			FieldsCondition: ty.MS{},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Gte.S("24h@h")
		logSearch.Range.Lte.S("now")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Equal(t, `index=nonprod message="this is a \"test\""`, requestBodyFields["search"])
	})

	t.Run("use last duration instead of explicit times", func(t *testing.T) {
		logSearch := &client.LogSearch{
			Fields:          ty.MS{"_": "hello"},
			FieldsCondition: ty.MS{},
			Options:         ty.MI{"index": "nonprod"},
		}
		logSearch.Range.Last.S("1min")

		requestBodyFields, err := getSearchRequest(logSearch, false)
		assert.NoError(t, err)
		assert.Equal(t, "-1min", requestBodyFields["earliest_time"])
		assert.Equal(t, "now", requestBodyFields["latest_time"])
	})
}
