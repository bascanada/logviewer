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

func TestReaderLogResult_parseLine(t *testing.T) {
	type fields struct {
		search                    *client.LogSearch
		kvRegexExtraction         *regexp.Regexp
		namedGroupRegexExtraction *regexp.Regexp
	}
	type args struct {
		line string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		wantEntry *client.LogEntry
	}{
		{
			name: "Test no filtering without regex",
			fields: fields{
				search: &client.LogSearch{
					Fields: ty.MS{
						"level": "info",
					},
				},
			},
			args: args{
				line: "this is a log line",
			},
			want: true,
			wantEntry: &client.LogEntry{
				Message: "this is a log line",
				Fields:  ty.MI{},
			},
		},
		{
			name: "Test filtering with regex",
			fields: fields{
				search: &client.LogSearch{
					Fields: ty.MS{
						"level": "info",
					},
				},
				namedGroupRegexExtraction: regexp.MustCompile(`(?P<level>info|error)`),
			},
			args: args{
				line: "this is a info log line",
			},
			want: true,
			wantEntry: &client.LogEntry{
				Message: "this is a info log line",
				Fields: ty.MI{
					"level": "info",
				},
			},
		},
		{
			name: "Test filtering with regex no match",
			fields: fields{
				search: &client.LogSearch{
					Fields: ty.MS{
						"level": "error",
					},
				},
				namedGroupRegexExtraction: regexp.MustCompile(`(?P<level>info|error)`),
			},
			args: args{
				line: "this is a info log line",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lr := &ReaderLogResult{
				search:                    tt.fields.search,
				kvRegexExtraction:         tt.fields.kvRegexExtraction,
				namedGroupRegexExtraction: tt.fields.namedGroupRegexExtraction,
				entries:                   []client.LogEntry{},
				fields:                    ty.UniSet[string]{},
			}
			if got := lr.parseLine(tt.args.line); got != tt.want {
				t.Errorf("ReaderLogResult.parseLine() = %v, want %v", got, tt.want)
			}
			if tt.wantEntry != nil {
				assert.Equal(t, tt.wantEntry.Message, lr.entries[0].Message)
				assert.Equal(t, tt.wantEntry.Fields, lr.entries[0].Fields)
			}
		})
	}
}
