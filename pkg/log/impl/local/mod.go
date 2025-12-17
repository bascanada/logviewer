package local

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"text/template"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/reader"
)

const (
	OptionsCmd   = "cmd"
	OptionsShell = "shell"

	defaultShellWindows    = "powershell"
	defaultShellArgWindows = "-Command"
	defaultShellUnix       = "sh"
	defaultShellArgUnix    = "-c"
)

type localLogClient struct{}

func getCommand(search *client.LogSearch) (string, error) {
	cmdTplStr := search.Options.GetString(OptionsCmd)

	if cmdTplStr == "" {
		return "", errors.New("cmd is missing for localLogClient")
	}

	tmpl, err := template.New("cmd").Parse(cmdTplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse command template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, search); err != nil {
		return "", fmt.Errorf("failed to execute command template: %w", err)
	}
	return buf.String(), nil
}

func (lc localLogClient) Get(ctx context.Context, search *client.LogSearch) (client.LogSearchResult, error) {
	cmdContent, err := getCommand(search)
	if err != nil {
		if err.Error() == "cmd is missing for localLogClient" {
			panic(err)
		}
		return nil, err
	}

	var shellName string
	var shellArgs []string

	if customShell, ok := search.Options.GetListOfStringsOk(OptionsShell); ok && len(customShell) > 0 {
		shellName = customShell[0]
		shellArgs = customShell[1:]
	} else {
		if runtime.GOOS == "windows" {
			shellName = defaultShellWindows
			shellArgs = []string{defaultShellArgWindows}
		} else {
			shellName = defaultShellUnix
			shellArgs = []string{defaultShellArgUnix}
		}
	}

	finalArgs := append(shellArgs, cmdContent)

	ecmd := exec.CommandContext(ctx, shellName, finalArgs...)

	stdout, err := ecmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = ecmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)

	return reader.GetLogResult(search, scanner, stdout)
}

func (lc localLogClient) GetFieldValues(ctx context.Context, search *client.LogSearch, fields []string) (map[string][]string, error) {
	// For local/text-based backends, we need to run a search and extract field values
	result, err := lc.Get(ctx, search)
	if err != nil {
		return nil, err
	}
	return client.GetFieldValuesFromResult(ctx, result, fields)
}

func GetLogClient() (client.LogClient, error) {
	return localLogClient{}, nil
}
