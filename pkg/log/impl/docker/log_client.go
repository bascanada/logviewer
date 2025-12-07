package docker

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	logclient "github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/reader"
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

	// Check if service is provided for service discovery
	if service := search.Options.GetString("service"); service != "" {
		// Use service discovery
		filterArgs := filters.NewArgs()
		filterArgs.Add("label", fmt.Sprintf("com.docker.compose.service=%s", service))

		// Optional project filter
		if project := search.Options.GetString("project"); project != "" {
			filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", project))
		}

		containers, err := lc.apiClient.ContainerList(ctx, container.ListOptions{
			Filters: filterArgs,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list containers for service %s: %w", service, err)
		}

		if len(containers) == 0 {
			return nil, fmt.Errorf("no running containers found for service %s", service)
		}

		// TODO: Use MultiLogSearchResult to merge logs from all containers when multiple replicas exist
		if len(containers) > 1 {
			fmt.Fprintf(os.Stderr, "WARN: Found %d containers for service '%s'. Showing logs for the first one (%s).\n", len(containers), service, containers[0].ID[:12])
		}

		// Use the first matching container
		containerID = containers[0].ID
	}

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

	follow := search.Follow

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

	return reader.GetLogResult(search, scanner, out)
}

func GetLogClient(host string) (logclient.LogClient, error) {
	// Prepare basic options
	opts := []client.Opt{
		client.FromEnv,
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	}

	// Try to get a connection helper (e.g., for ssh://)
	helper, err := connhelper.GetConnectionHelper(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection helper: %w", err)
	}

	// If a helper is found (SSH case), inject its DialContext
	// This allows using the system ssh binary and .ssh/config file
	if helper != nil {
		opts = append(opts, client.WithDialContext(helper.Dialer))
	}

	apiClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		// Il est préférable de retourner l'erreur plutôt que de panic
		return nil, err
	}

	return DockerLogClient{
		apiClient: apiClient,
		host:      host,
	}, nil
}
