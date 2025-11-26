package printer

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"text/template"

	"github.com/bascanada/logviewer/pkg/log/client"
)

type LogPrinter interface {
	Display(ctx context.Context, result client.LogSearchResult) error
}

func WrapIoWritter(ctx context.Context, result client.LogSearchResult, writer io.Writer, update func()) (bool, error) {

	printerOptions := result.GetSearch().PrinterOptions

	templateConfig := printerOptions.Template

	if templateConfig.Value == "" {
		templateConfig.S("[{{.Timestamp.Format \"15:04:05\" }}] [{{.ContextID}}] {{.Level}} {{.Message}}")
	}

	tmpl, err := template.New("print_printer").Funcs(GetTemplateFunctionsMap()).Parse(templateConfig.Value + "\n")
	if err != nil {
		return false, err
	}

	// Prepare messageRegex if present
	var messageRegex *regexp.Regexp
	if printerOptions.MessageRegex.Set && printerOptions.MessageRegex.Value != "" {
		var errRegex error
		messageRegex, errRegex = regexp.Compile(printerOptions.MessageRegex.Value)
		if errRegex != nil {
			return false, errRegex
		}
	}


	entries, newEntriesChannel, err := result.GetEntries(ctx)
	if err != nil {
		return false, err
	}

	if err := processEntries(writer, tmpl, messageRegex, entries); err != nil {
		return false, err
	}

	update()

	if newEntriesChannel != nil {
		go func() {
			update()
			for entries := range newEntriesChannel {
				if len(entries) > 0 {
					if err := processEntries(writer, tmpl, messageRegex, entries); err != nil {
						fmt.Fprintf(os.Stderr, "error printing log entries: %v\n", err)
					}
					update()
				}
			}
		}()
	}

	// new goroutine to listen for errors
	if errChan := result.Err(); errChan != nil {
		go func() {
			for err := range errChan {
				fmt.Fprintf(os.Stderr, "an error occurred: %v\n", err)
			}
		}()
	}

	return newEntriesChannel != nil, nil
}

func processEntries(writer io.Writer, tmpl *template.Template, messageRegex *regexp.Regexp, entries []client.LogEntry) error {
	for i, entry := range entries {
		if messageRegex != nil {
			matches := messageRegex.FindStringSubmatch(entry.Message)
			if len(matches) > 1 {
				entries[i].Message = matches[1]
			}
		}
		if err := tmpl.Execute(writer, entries[i]); err != nil {
			return err
		}
	}
	return nil
}
