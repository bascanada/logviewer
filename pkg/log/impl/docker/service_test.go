package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"testing"

	logclient "github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	args := m.Called(ctx, options)
	return args.Get(0).([]types.Container), args.Error(1)
}

func (m *MockDockerClient) ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, container, options)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockDockerClient) Ping(ctx context.Context) (types.Ping, error) {
	args := m.Called(ctx)
	return args.Get(0).(types.Ping), args.Error(1)
}

func makeLogFrame(msg string) []byte {
	header := make([]byte, 8)
	header[0] = 1 // stdout
	binary.BigEndian.PutUint32(header[4:], uint32(len(msg)))
	return append(header, []byte(msg)...)
}

func TestServiceLogs(t *testing.T) {
	mockClient := new(MockDockerClient)
	lc := DockerLogClient{
		apiClient: mockClient,
		host:      "local",
	}

	ctx := context.Background()
	search := &logclient.LogSearch{
		Options: ty.MI{
			"service": "web-app",
		},
		FieldExtraction: logclient.FieldExtraction{
			TimestampRegex: ty.Opt[string]{},
		},
	}

	// 1. Expect ContainerList call
	mockClient.On("ContainerList", ctx, mock.MatchedBy(func(opts container.ListOptions) bool {
		// Verify filters
		return opts.Filters.ExactMatch("label", "com.docker.compose.service=web-app")
	})).Return([]types.Container{
		{ID: "container_id_1_long", Names: []string{"/web-app-1"}},
		{ID: "container_id_2_long", Names: []string{"/web-app-2"}},
	}, nil)

	// 2. Expect ContainerLogs calls for both containers
	logContent1 := makeLogFrame("2024-01-01T00:00:01.000000000Z log from c1\n")
	logContent2 := makeLogFrame("2024-01-01T00:00:02.000000000Z log from c2\n")

	mockClient.On("ContainerLogs", ctx, "container_id_1_long", mock.Anything).Return(io.NopCloser(bytes.NewReader(logContent1)), nil)
	mockClient.On("ContainerLogs", ctx, "container_id_2_long", mock.Anything).Return(io.NopCloser(bytes.NewReader(logContent2)), nil)

	// Execute
	result, err := lc.Get(ctx, search)
	assert.NoError(t, err)

	// Verify we got a MultiLogSearchResult
	// We can check if it implements specific interface or just check behavior
	
	// Get entries
	entries, _, err := result.GetEntries(ctx)
	assert.NoError(t, err)

	// We should get 2 entries
	assert.Len(t, entries, 2)

	// Sort by timestamp is handled by MultiLogSearchResult
	// Note: MultiLogSearchResult sorts by timestamp.
	// 00:00:01 comes before 00:00:02
	assert.Equal(t, " log from c1", entries[0].Message)
	assert.Equal(t, " log from c2", entries[1].Message)

	// Verify ContextID is set correctly
	// The ContextID is set to the first 12 characters of the container ID
	assert.Equal(t, "container_id", entries[0].ContextID)
	assert.Equal(t, "container_id", entries[1].ContextID)

	mockClient.AssertExpectations(t)
}

func TestServiceLogs_SingleContainer(t *testing.T) {
	mockClient := new(MockDockerClient)
	lc := DockerLogClient{
		apiClient: mockClient,
		host:      "local",
	}

	ctx := context.Background()
	search := &logclient.LogSearch{
		Options: ty.MI{
			"service": "web-app",
		},
		FieldExtraction: logclient.FieldExtraction{
			TimestampRegex: ty.Opt[string]{},
		},
	}

	// 1. Expect ContainerList call returning 1 container
	mockClient.On("ContainerList", ctx, mock.Anything).Return([]types.Container{
		{ID: "c1", Names: []string{"/web-app-1"}},
	}, nil)

	// 2. Expect ContainerLogs call for the single container
	logContent := makeLogFrame("2024-01-01T00:00:01.000000000Z single log\n")
	mockClient.On("ContainerLogs", ctx, "c1", mock.Anything).Return(io.NopCloser(bytes.NewReader(logContent)), nil)

	// Execute
	result, err := lc.Get(ctx, search)
	assert.NoError(t, err)

	entries, _, err := result.GetEntries(ctx)
	assert.NoError(t, err)

	assert.Len(t, entries, 1)
	assert.Equal(t, " single log", entries[0].Message)

	mockClient.AssertExpectations(t)
}
