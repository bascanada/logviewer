package docker

import (
	"bufio"
	"context"
	"fmt"

	logclient "github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/reader"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const regexDockerTimestamp = "(([0-9]*)-([0-9]*)-([0-9]*)T([0-9]*):([0-9]*):([0-9]*).([0-9]*)Z)"

type DockerLogClient struct {
	apiClient *client.Client
	host      string
}

func (lc DockerLogClient) Get(ctx context.Context, search *logclient.LogSearch) (logclient.LogSearchResult, error) {

	if search.FieldExtraction.TimestampRegex.Set == false {
		search.FieldExtraction.TimestampRegex.S(regexDockerTimestamp)
	}

	// Specify the container ID or name
	containerID := search.Options.GetString("container")

	var since, until string

	if search.Range.Last.Value != "" {
		since = search.Range.Last.Value
	} else {
		if search.Range.Gte.Value != "" {
			since = search.Range.Gte.Value
		}

		if search.Range.Lte.Value != "" {
			until = search.Range.Lte.Value
		}
	}

	tail := "all"

	if search.Size.Set {
		tail = fmt.Sprintf("%d", search.Size.Value)
	}

	follow := search.Refresh.Follow.Value

	options := container.LogsOptions{
		ShowStdout: search.Options.GetOr("showStdout", true).(bool),
		ShowStderr: search.Options.GetOr("showStderr", true).(bool),
		Timestamps: search.Options.GetOr("timestamps", true).(bool),
		Details:    search.Options.GetOr("details", false).(bool),
		Since:      since,
		Until:      until,
		Follow:     follow,
		Tail:       tail,
	}
	out, err := lc.apiClient.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(out)

	return reader.GetLogResult(search, scanner, out), nil
}

func GetLogClient(host string) (logclient.LogClient, error) {

	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	return DockerLogClient{
		apiClient: apiClient,
		host:      host,
	}, nil
}
