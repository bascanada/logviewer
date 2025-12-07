package docker

import (
	"bufio"
	"context"
	"fmt"

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
	// Préparation des options de base
	opts := []client.Opt{
		client.FromEnv,
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	}

	// Tenter de récupérer un helper de connexion (ex: pour ssh://)
	helper, err := connhelper.GetConnectionHelper(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection helper: %w", err)
	}

	// Si un helper est trouvé (cas du SSH), on injecte son DialContext
	// C'est ce qui permet d'utiliser le binaire ssh système et le fichier .ssh/config
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
