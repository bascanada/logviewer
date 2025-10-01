package printer

import (
	"context"
	"io"
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
		templateConfig.S("[{{.Timestamp.Format \"15:04:05\" }}] {{.Level}} {{.Message}}")
	}

	tmpl, err3 := template.New("print_printer").Funcs(GetTemplateFunctionsMap()).Parse(templateConfig.Value + "\n")
	if err3 != nil {
		return false, err3
	}

	// Prepare messageRegex if present
	var messageRegex *regexp.Regexp
	if printerOptions.MessageRegex.Set && printerOptions.MessageRegex.Value != "" {
		var err error
		messageRegex, err = regexp.Compile(printerOptions.MessageRegex.Value)
		if err != nil {
			return false, err
		}
	}

	entries, newEntriesChannel, err := result.GetEntries(ctx)
	if err != nil {
		return false, err
	}

	for i, entry := range entries {
		if messageRegex != nil {
			matches := messageRegex.FindStringSubmatch(entry.Message)
			if len(matches) > 1 {
				entries[i].Message = matches[1]
			}
		}
		tmpl.Execute(writer, entries[i])
	}

	update()

	if err != nil {
		return false, err
	}

	if newEntriesChannel != nil {
		go func() {
			update()
			for entries := range newEntriesChannel {
				if len(entries) > 0 {
					for i, entry := range entries {
						if messageRegex != nil {
							matches := messageRegex.FindStringSubmatch(entry.Message)
							if len(matches) > 1 {
								entries[i].Message = matches[1]
							}
						}
						tmpl.Execute(writer, entries[i])
					}
					update()
				}
			}
		}()
	}

	return newEntriesChannel != nil, nil
}
