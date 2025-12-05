package reader

import (
	"bufio"
	"context"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
)

const maxBatchSize = 10

type ReaderLogResult struct {
	search  *client.LogSearch
	scanner *bufio.Scanner
	closer  io.Closer

	// mutex because updated by goroutine
	entries []client.LogEntry
	fields  ty.UniSet[string]

	kvRegexExtraction         *regexp.Regexp
	namedGroupRegexExtraction *regexp.Regexp
	regexDate                 *regexp.Regexp

	ErrChan chan error
}

func (lr ReaderLogResult) Err() <-chan error {
	return lr.ErrChan
}

func (lr ReaderLogResult) GetSearch() *client.LogSearch {
	return lr.search
}

func (lr *ReaderLogResult) parseLine(line string) bool {
	entry := client.LogEntry{
		Message: line,
		Fields:  make(ty.MI),
	}

	// check if we have a date at the beginning and parse / remove it
	if lr.regexDate != nil {
		entry.Message = strings.TrimLeft(lr.regexDate.ReplaceAllStringFunc(line, func(v string) string {
			// Try parsing using the configured format (typically RFC3339),
			// then fallback to common space-separated layouts used in logs.
			var parsed time.Time
			var err error

			parsed, err = time.Parse(ty.Format, v)
			if err != nil {
				// Try space-separated layout with milliseconds in local timezone
				parsed, err = time.ParseInLocation("2006-01-02 15:04:05.000", v, time.Local)
			}
			if err != nil {
				// Try without milliseconds
				parsed, err = time.ParseInLocation("2006-01-02 15:04:05", v, time.Local)
			}
			if err == nil {
				entry.Timestamp = parsed
			}
			return ""
		}), " ")
	}

	if lr.namedGroupRegexExtraction != nil {
		match := lr.namedGroupRegexExtraction.FindStringSubmatch(line)
		if len(match) > 0 {
			for i, name := range lr.namedGroupRegexExtraction.SubexpNames() {
				if i != 0 && name != "" {
					trimmedValue := strings.TrimSpace(match[i])
					lr.fields.Add(name, trimmedValue)
					entry.Fields[name] = trimmedValue
				}
			}
		}
	}

	if lr.kvRegexExtraction != nil {
		matches := lr.kvRegexExtraction.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				trimmedKey := strings.TrimSpace(match[1])
				trimmedValue := strings.TrimSpace(match[2])
				lr.fields.Add(trimmedKey, trimmedValue)
				entry.Fields[trimmedKey] = trimmedValue
			}
		}
	}

	if lr.namedGroupRegexExtraction != nil || lr.kvRegexExtraction != nil {
		for k, v := range lr.search.Fields {
			if vv, ok := entry.Fields[k]; ok {
				if v != vv {
					return false
				}
			} else {
				return false
			}
		}
	}

	// Try both lowercase and uppercase versions for Level field
	if level := entry.Fields.GetString("level"); level != "" {
		entry.Level = level
	} else if level := entry.Fields.GetString("Level"); level != "" {
		entry.Level = level
	}
	lr.entries = append(lr.entries, entry)
	return true
}

func (lr *ReaderLogResult) loadEntries() bool {
	lr.entries = make([]client.LogEntry, 0)

	for lr.scanner.Scan() {
		line := lr.scanner.Text()
		lr.parseLine(line)
	}
	return len(lr.entries) > 0
}

func (lr ReaderLogResult) GetEntries(ctx context.Context) ([]client.LogEntry, chan []client.LogEntry, error) {

	if !lr.search.Follow {
		lr.loadEntries()
		lr.closer.Close()
		return lr.entries, nil, nil
	} else {
		c := make(chan []client.LogEntry)

		go func() {
			defer close(c)
			defer lr.closer.Close()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					{
						if lr.scanner.Scan() {
							if lr.parseLine(lr.scanner.Text()) {
								c <- []client.LogEntry{lr.entries[len(lr.entries)-1]}
							}
						}
					}
				}
			}
		}()

		return []client.LogEntry{}, c, nil
	}
}

func (lr ReaderLogResult) GetFields(ctx context.Context) (ty.UniSet[string], chan ty.UniSet[string], error) {
	return lr.fields, nil, nil
}

func (lr ReaderLogResult) GetPaginationInfo() *client.PaginationInfo {
	return nil
}

func GetLogResult(
	search *client.LogSearch,
	scanner *bufio.Scanner,
	closer io.Closer,
) (*ReaderLogResult, error) {

	var namedGroupRegexExtraction *regexp.Regexp
	if search.FieldExtraction.GroupRegex.Value != "" {
		var err error
		namedGroupRegexExtraction, err = regexp.Compile(search.FieldExtraction.GroupRegex.Value)
		if err != nil {
			return nil, err
		}
	}

	var kvRegexExtraction *regexp.Regexp
	if search.FieldExtraction.KvRegex.Value != "" {
		var err error
		kvRegexExtraction, err = regexp.Compile(search.FieldExtraction.KvRegex.Value)
		if err != nil {
			return nil, err
		}
	}

	var regexDateExtraction *regexp.Regexp
	if search.FieldExtraction.TimestampRegex.Value != "" {
		var err error
		regexDateExtraction, err = regexp.Compile(search.FieldExtraction.TimestampRegex.Value)
		if err != nil {
			return nil, err
		}
	}

	result := &ReaderLogResult{
		search:                    search,
		scanner:                   scanner,
		closer:                    closer,
		namedGroupRegexExtraction: namedGroupRegexExtraction,
		kvRegexExtraction:         kvRegexExtraction,
		regexDate:                 regexDateExtraction,
		fields:                    make(ty.UniSet[string]),
	}

	return result, nil
}
