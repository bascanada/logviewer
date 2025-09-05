package reader

import (
	"regexp"
	"testing"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

func TestTimestampExtraction(t *testing.T) {

	logResult := ReaderLogResult{
		entries: make([]client.LogEntry, 0),
		search: &client.LogSearch{
			Fields: ty.MS{},
		},
		fields: ty.UniSet[string]{},

		regexDate: regexp.MustCompile(ty.RegexTimestampFormat),
	}

	expectedTime, _ := time.Parse(ty.Format, "2024-06-24T15:27:29.669455265Z")
	isParsed := logResult.parseLine("\x01\x00\x00\x00\x00\x00\x00\x802024-06-24T15:27:29.669455265Z /docker-entrypoint.sh: /docker-entrypoint.d/ is not empty, will attempt to perform configuration")
	entry := logResult.entries[0]

	assert.Equal(t, true, isParsed)
	assert.Equal(t, "\x01\x00\x00\x00\x00\x00\x00\x80 /docker-entrypoint.sh: /docker-entrypoint.d/ is not empty, will attempt to perform configuration", entry.Message)
	assert.Equal(t, expectedTime, entry.Timestamp)

}

func TestReaderLogResult_GetPaginationInfo(t *testing.T) {
	result := ReaderLogResult{}
	assert.Nil(t, result.GetPaginationInfo())
}
