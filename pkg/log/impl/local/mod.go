package local

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"text/template"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/reader"
)

const (
	OptionsCmd = "cmd"
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
	cmd, err := getCommand(search)
	if err != nil {
		if err.Error() == "cmd is missing for localLogClient" {
			panic(err)
		}
		return nil, err
	}

	ecmd := exec.CommandContext(ctx, "sh", "-c", cmd)

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

func GetLogClient() (client.LogClient, error) {
	return localLogClient{}, nil
}
