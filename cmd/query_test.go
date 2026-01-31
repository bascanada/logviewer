package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

func TestRunQueryValues(t *testing.T) {
	mockClient := &client.MockLogClient{
		OnValues: func(search client.LogSearch, field string) ([]string, error) {
			if field == "level" {
				return []string{"INFO", "ERROR"}, nil
			}
			if field == "app" {
				return []string{"frontend", "backend"}, nil
			}
			return []string{}, nil
		},
	}

	search := client.LogSearch{
		Fields: ty.MS{"env": "prod"},
	}

	t.Run("text output", func(t *testing.T) {
		var buf bytes.Buffer
		err := RunQueryValues(&buf, mockClient, search, []string{"level"}, false)
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "level")
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "ERROR")
	})

	t.Run("json output", func(t *testing.T) {
		var buf bytes.Buffer
		err := RunQueryValues(&buf, mockClient, search, []string{"app"}, true)
		assert.NoError(t, err)

		var result map[string][]string
		err = json.Unmarshal(buf.Bytes(), &result)
		assert.NoError(t, err)

		assert.Contains(t, result, "app")
		assert.Equal(t, []string{"frontend", "backend"}, result["app"])
	})

	t.Run("multiple fields", func(t *testing.T) {
		var buf bytes.Buffer
		err := RunQueryValues(&buf, mockClient, search, []string{"level", "app"}, false)
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "level")
		assert.Contains(t, output, "app")
	})
}

func TestRunQueryField(t *testing.T) {
	mockClient := &client.MockLogClient{
		OnFields: func(search client.LogSearch) (map[string][]string, error) {
			return map[string][]string{
				"level":   {"INFO", "WARN"},
				"message": {"foo bar"},
			}, nil
		},
	}

	search := client.LogSearch{}

	t.Run("displays fields and examples", func(t *testing.T) {
		var buf bytes.Buffer
		err := RunQueryField(&buf, mockClient, search)
		assert.NoError(t, err)

		output := buf.String()
		// Check for field name
		assert.True(t, strings.Contains(output, "level") || strings.Contains(output, "level "), "output should contain 'level'")
		// Check for example value
		assert.True(t, strings.Contains(output, "INFO") || strings.Contains(output, "INFO\n"), "output should contain 'INFO'")
		assert.Contains(t, output, "message")
		assert.Contains(t, output, "foo bar")
	})
}
