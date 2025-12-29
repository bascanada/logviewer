package reader

import (
	"bufio"
	"context"
	"fmt"
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

func (lr *ReaderLogResult) parseBlock(block string) (*client.LogEntry, bool) {
	// Split into first line and rest
	var firstLine string
	var rest string
	if idx := strings.Index(block, "\n"); idx != -1 {
		firstLine = block[:idx]
		rest = block[idx+1:]
	} else {
		firstLine = block
	}

	entry := client.LogEntry{
		Message: firstLine,
		Fields:  make(ty.MI),
	}

	// check if we have a date (anywhere in the line) and parse / remove it.
	// When logs are produced via SSH they can include extra prefixes
	// (e.g. PTY markers) before the timestamp. Find the timestamp match,
	// parse it and then remove everything up to the end of the match so the
	// resulting message doesn't keep the prefix.
	if lr.regexDate != nil {
		if loc := lr.regexDate.FindStringIndex(firstLine); loc != nil {
			matched := firstLine[loc[0]:loc[1]]
			if parsed, err := parseTimestamp(matched); err == nil {
				entry.Timestamp = parsed
			}
			// Preserve any prefix bytes that appear before the timestamp
			// (e.g., PTY markers). Keep the remainder after the timestamp
			// as-is so control characters are not lost.
			prefix := firstLine[:loc[0]]
			if loc[1] < len(firstLine) {
				rest := firstLine[loc[1]:]
				entry.Message = prefix + rest
			} else {
				entry.Message = prefix
			}
		} else {
			entry.Message = strings.TrimSpace(firstLine)
		}
	}

	// Extract JSON fields using shared function
	client.ExtractJSONFromEntry(&entry, lr.search)

	// Update field set for discovery
	if lr.search.FieldExtraction.Json.Value {
		for k, v := range entry.Fields {
			lr.fields.Add(k, fmt.Sprintf("%v", v))
		}
	}

	if lr.namedGroupRegexExtraction != nil {
		match := lr.namedGroupRegexExtraction.FindStringSubmatch(firstLine)
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
		matches := lr.kvRegexExtraction.FindAllStringSubmatch(firstLine, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				trimmedKey := strings.TrimSpace(match[1])
				trimmedValue := strings.TrimSpace(match[2])
				lr.fields.Add(trimmedKey, trimmedValue)
				entry.Fields[trimmedKey] = trimmedValue
			}
		}
	}

	// Try both lowercase and uppercase versions for Level field
	// (must happen before filter check so entry.Level is populated)
	if level := entry.Fields.GetString("level"); level != "" {
		entry.Level = level
	} else if level := entry.Fields.GetString("Level"); level != "" {
		entry.Level = level
	}

	// Check if results are pre-filtered (e.g., by hl locally)
	// When __preFiltered__ is true, skip client-side filtering entirely
	// Note: __hybridHL__ is NOT used to skip filtering because we can't know
	// if hl actually ran on the remote until after reading all output.
	// For hybrid SSH mode, we always filter client-side to ensure correctness.
	isPreFiltered := lr.search.Options.GetBool("__preFiltered__")

	// Apply filter using the new recursive filter system
	// Skip filtering only if explicitly pre-filtered (local hl mode)
	if !isPreFiltered {
		if lr.namedGroupRegexExtraction != nil || lr.kvRegexExtraction != nil || lr.search.FieldExtraction.Json.Value {
			effectiveFilter := lr.search.GetEffectiveFilter()
			if effectiveFilter != nil {
				if !effectiveFilter.Match(entry) {
					return nil, false
				}
			}
		}
	}

	if rest != "" {
		entry.Message = entry.Message + "\n" + rest
	}

	return &entry, true
}

func (lr *ReaderLogResult) processLine(line string, pendingBlock *strings.Builder, onEntry func(client.LogEntry)) {
	// Consider a line as a new entry when no timestamp regex is configured,
	// or when the configured timestamp regex matches anywhere in the line.
	// Some log producers (or PTY vs non-PTY SSH outputs) prefix lines with
	// extra markers before the timestamp, so requiring the timestamp to be
	// at index 0 is too strict and breaks multiline detection.
	isNewEntry := true
	if lr.regexDate != nil {
		isNewEntry = lr.regexDate.MatchString(line)
	}

	if isNewEntry {
		lr.flushBlock(pendingBlock, onEntry)
		pendingBlock.WriteString(line)
	} else {
		if pendingBlock.Len() > 0 {
			pendingBlock.WriteString("\n")
		}
		pendingBlock.WriteString(line)
	}
}

func (lr *ReaderLogResult) flushBlock(pendingBlock *strings.Builder, onEntry func(client.LogEntry)) {
	if pendingBlock.Len() > 0 {
		if entry, ok := lr.parseBlock(pendingBlock.String()); ok {
			onEntry(*entry)
		}
		pendingBlock.Reset()
	}
}

func (lr *ReaderLogResult) loadEntries() bool {
	lr.entries = make([]client.LogEntry, 0)
	var pendingBlock strings.Builder

	onEntry := func(entry client.LogEntry) {
		lr.entries = append(lr.entries, entry)
	}

	for lr.scanner.Scan() {
		lr.processLine(lr.scanner.Text(), &pendingBlock, onEntry)
	}
	lr.flushBlock(&pendingBlock, onEntry)

	return len(lr.entries) > 0
}

func (lr *ReaderLogResult) GetEntries(ctx context.Context) ([]client.LogEntry, chan []client.LogEntry, error) {

	if !lr.search.Follow {
		lr.loadEntries()
		lr.closer.Close()
		return lr.entries, nil, nil
	} else {
		c := make(chan []client.LogEntry)

		go func() {
			defer close(c)
			defer lr.closer.Close()

			var pendingBlock strings.Builder
			onEntry := func(entry client.LogEntry) {
				c <- []client.LogEntry{entry}
			}

			for {
				select {
				case <-ctx.Done():
					return
				default:
					{
						if lr.scanner.Scan() {
							lr.processLine(lr.scanner.Text(), &pendingBlock, onEntry)
						} else {
							// EOF or error
							lr.flushBlock(&pendingBlock, onEntry)
							return
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
		// Allow timestamp regexes anchored at start (^) to still match when
		// lines contain prefixes (e.g., SSH/PTY markers). To be forgiving for
		// common user patterns, compile an unanchored version for detection
		// and extraction by removing a leading '^' if present.
		pattern := search.FieldExtraction.TimestampRegex.Value
		if strings.HasPrefix(pattern, "^") {
			pattern = strings.TrimPrefix(pattern, "^")
		}
		regexDateExtraction, err = regexp.Compile(pattern)
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

func parseTimestamp(v interface{}) (time.Time, error) {
	var parsed time.Time
	var err error

	switch t := v.(type) {
	case string:
		parsed, err = time.Parse(ty.Format, t)
		if err != nil {
			parsed, err = time.ParseInLocation("2006-01-02 15:04:05.000", t, time.Local)
		}
		if err != nil {
			parsed, err = time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
		}
	case float64:
		sec := int64(t)
		nsec := int64((t - float64(sec)) * 1e9)
		parsed = time.Unix(sec, nsec)
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp format: %T", v)
	}

	return parsed, err
}
