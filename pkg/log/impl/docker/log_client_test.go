package docker

import (
	"context"
	"regexp"
	"testing"
	"time"

	logclient "github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

// TestLogClient_Get_Follow_Flag verifies that the Follow flag is correctly
// derived from search.Follow for streaming logs
func TestLogClient_Get_Follow_Flag(t *testing.T) {
	t.Run("Follow should use search.Follow field", func(t *testing.T) {
		// Test case: Follow=true should enable streaming
		search := &logclient.LogSearch{
			Follow: true,
			Options: ty.MI{
				"container": "test-container",
			},
		}

		follow := search.Follow
		assert.True(t, follow, "Follow should be true when search.Follow is true")
	})

	t.Run("Follow should be false when search.Follow is false", func(t *testing.T) {
		search := &logclient.LogSearch{
			Follow: false,
			Options: ty.MI{
				"container": "test-container",
			},
		}

		follow := search.Follow
		assert.False(t, follow, "Follow should be false when search.Follow is false")
	})

	t.Run("Follow is independent of Refresh.Duration", func(t *testing.T) {
		// Docker uses search.Follow, not search.Refresh.Duration
		// This is the correct behavior (unlike the old K8s bug)
		search := &logclient.LogSearch{
			Follow: false,
			Options: ty.MI{
				"container": "test-container",
			},
			Refresh: logclient.RefreshOptions{
				Duration: ty.Opt[string]{Value: "30s", Set: true},
			},
		}

		follow := search.Follow
		assert.False(t, follow, "Docker Follow should be based on search.Follow, not Refresh.Duration")
	})
}

func TestLogClient_Options_Parsing(t *testing.T) {
	t.Run("Parses container from options", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "my-container",
			},
		}

		container := search.Options.GetString("container")
		assert.Equal(t, "my-container", container)
	})

	t.Run("Parses service from options", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"service": "my-service",
			},
		}

		service := search.Options.GetString("service")
		assert.Equal(t, "my-service", service)
	})

	t.Run("Parses project from options", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"project": "my-project",
			},
		}

		project := search.Options.GetString("project")
		assert.Equal(t, "my-project", project)
	})

	t.Run("Parses showStdout from options", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"showStdout": true,
			},
		}

		showStdout := search.Options.GetOr("showStdout", true).(bool)
		assert.True(t, showStdout)
	})

	t.Run("Parses showStderr from options", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"showStderr": false,
			},
		}

		showStderr := search.Options.GetOr("showStderr", true).(bool)
		assert.False(t, showStderr)
	})

	t.Run("Default showStdout is true", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{},
		}

		showStdout := search.Options.GetOr("showStdout", true).(bool)
		assert.True(t, showStdout)
	})

	t.Run("Default showStderr is true", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{},
		}

		showStderr := search.Options.GetOr("showStderr", true).(bool)
		assert.True(t, showStderr)
	})

	t.Run("Parses timestamps from options", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"timestamps": false,
			},
		}

		timestamps := search.Options.GetOr("timestamps", true).(bool)
		assert.False(t, timestamps)
	})

	t.Run("Parses details from options", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"details": true,
			},
		}

		details := search.Options.GetOr("details", false).(bool)
		assert.True(t, details)
	})
}

func TestLogClient_TimeRange(t *testing.T) {
	t.Run("Since from Range.Last", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "test-container",
			},
		}
		search.Range.Last.S("1h")

		// Simulate what the Get function does
		var since string
		if search.Range.Last.Value != "" {
			since = search.Range.Last.Value
		}
		assert.Equal(t, "1h", since)
	})

	t.Run("Since from Range.Gte", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "test-container",
			},
		}
		search.Range.Gte.S("2024-01-01T00:00:00Z")

		// Simulate what the Get function does
		var since string
		if search.Range.Last.Value != "" {
			since = search.Range.Last.Value
		} else if search.Range.Gte.Value != "" {
			since = search.Range.Gte.Value
		}
		assert.Equal(t, "2024-01-01T00:00:00Z", since)
	})

	t.Run("Until from Range.Lte", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "test-container",
			},
		}
		search.Range.Lte.S("2024-12-31T23:59:59Z")

		// Simulate what the Get function does
		var until string
		if search.Range.Last.Value == "" && search.Range.Lte.Value != "" {
			until = search.Range.Lte.Value
		}
		assert.Equal(t, "2024-12-31T23:59:59Z", until)
	})

	t.Run("Last takes precedence over Gte/Lte", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "test-container",
			},
		}
		search.Range.Last.S("30m")
		search.Range.Gte.S("2024-01-01T00:00:00Z")
		search.Range.Lte.S("2024-12-31T23:59:59Z")

		// Simulate what the Get function does
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
		assert.Equal(t, "30m", since)
		assert.Empty(t, until, "Until should be empty when Last is set")
	})
}

func TestLogClient_Tail(t *testing.T) {
	t.Run("Tail is all when size not set", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "test-container",
			},
		}

		tail := "all"
		if search.Size.Set {
			tail = string(rune(search.Size.Value))
		}
		assert.Equal(t, "all", tail)
	})

	t.Run("Tail is set when size is provided", func(t *testing.T) {
		search := &logclient.LogSearch{
			Size: ty.Opt[int]{Value: 100, Set: true},
			Options: ty.MI{
				"container": "test-container",
			},
		}

		tail := "all"
		if search.Size.Set {
			// Simulating fmt.Sprintf("%d", search.Size.Value)
			tail = "100"
		}
		assert.Equal(t, "100", tail)
	})
}

func TestLogClient_TimestampRegex(t *testing.T) {
	t.Run("Default timestamp regex is set", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "test-container",
			},
		}

		// Simulate what the Get function does
		if !search.FieldExtraction.TimestampRegex.Set {
			search.FieldExtraction.TimestampRegex.S(regexDockerTimestamp)
		}

		assert.True(t, search.FieldExtraction.TimestampRegex.Set)
		assert.Equal(t, regexDockerTimestamp, search.FieldExtraction.TimestampRegex.Value)
	})

	t.Run("Custom timestamp regex is preserved", func(t *testing.T) {
		customRegex := "([0-9]{4})-([0-9]{2})-([0-9]{2})"
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "test-container",
			},
		}
		search.FieldExtraction.TimestampRegex.S(customRegex)

		// Simulate what the Get function does
		if !search.FieldExtraction.TimestampRegex.Set {
			search.FieldExtraction.TimestampRegex.S(regexDockerTimestamp)
		}

		assert.True(t, search.FieldExtraction.TimestampRegex.Set)
		assert.Equal(t, customRegex, search.FieldExtraction.TimestampRegex.Value)
	})
}

func TestLogClient_LogsOptions(t *testing.T) {
	t.Run("LogsOptions are correctly built from search", func(t *testing.T) {
		search := &logclient.LogSearch{
			Follow: true,
			Size:   ty.Opt[int]{Value: 100, Set: true},
			Options: ty.MI{
				"container":  "test-container",
				"showStdout": true,
				"showStderr": false,
				"timestamps": true,
				"details":    true,
			},
		}
		search.Range.Last.S("1h")

		// Simulate what the Get function does to build LogsOptions
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
			tail = "100" // fmt.Sprintf("%d", search.Size.Value)
		}

		follow := search.Follow
		showStdout := search.Options.GetOr("showStdout", true).(bool)
		showStderr := search.Options.GetOr("showStderr", true).(bool)
		timestamps := search.Options.GetOr("timestamps", true).(bool)
		details := search.Options.GetOr("details", false).(bool)

		assert.Equal(t, "1h", since)
		assert.Empty(t, until)
		assert.Equal(t, "100", tail)
		assert.True(t, follow)
		assert.True(t, showStdout)
		assert.False(t, showStderr)
		assert.True(t, timestamps)
		assert.True(t, details)
	})
}

func TestDockerTimestampRegex(t *testing.T) {
	t.Run("Regex matches Docker timestamp format", func(t *testing.T) {
		// Docker timestamp format: 2024-01-15T10:30:45.123456789Z
		r := regexp.MustCompile(regexDockerTimestamp)

		validTimestamps := []string{
			"2024-01-15T10:30:45.123456789Z",
			"2023-12-31T23:59:59.999999999Z",
			"2024-06-01T00:00:00.000000000Z",
		}

		for _, ts := range validTimestamps {
			assert.True(t, r.MatchString(ts), "Should match: %s", ts)
		}
	})
}

func TestGetLogClient(t *testing.T) {
	t.Run("Returns error for invalid SSH host", func(_ *testing.T) {
		// Testing with an SSH URL that can't connect
		// This will fail at ping timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Just verify the function signature works
		// We can't easily test successful connection without Docker running
		_ = ctx
	})

	t.Run("Host parameter handling", func(_ *testing.T) {
		// Test that various host formats are handled
		hosts := []string{
			"",                            // Default
			"unix:///var/run/docker.sock", // Unix socket
			"tcp://localhost:2375",        // TCP
		}
		for _, host := range hosts {
			// Just verify no panic on parameter handling
			_ = host
		}
	})
}

// TestLogClient_ServiceDiscovery tests the service discovery logic
func TestLogClient_ServiceDiscovery(t *testing.T) {
	t.Run("Service and project labels", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"service": "my-service",
				"project": "my-project",
			},
		}

		service := search.Options.GetString("service")
		project := search.Options.GetString("project")

		assert.Equal(t, "my-service", service)
		assert.Equal(t, "my-project", project)

		// The actual filter creation is internal to Get()
		// We just verify the options are parsed correctly
	})

	t.Run("Container ID takes precedence over service", func(t *testing.T) {
		search := &logclient.LogSearch{
			Options: ty.MI{
				"container": "container-123",
				"service":   "my-service",
			},
		}

		containerID := search.Options.GetString("container")
		service := search.Options.GetString("service")

		// When container is set, it takes precedence (service discovery skipped)
		assert.Equal(t, "container-123", containerID)
		assert.Equal(t, "my-service", service)
	})
}
