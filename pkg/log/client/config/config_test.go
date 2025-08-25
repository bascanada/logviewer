package config

import (
	"testing"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSearchContext_VariableSubstitution(t *testing.T) {
	// Setup environment variables
	t.Setenv("ENV_HOST", "prod.server.com")

	// Setup config
	cfg := ContextConfig{
		Contexts: Contexts{
			"test-context": {
				Client: "test-client",
				Search: client.LogSearch{
					Fields: ty.MS{
						"hostname": "${host}",
						"service":  "api",
						"env_host": "$ENV_HOST",
					},
					Options: ty.MI{
						"endpoint": "https://log-api/${host}",
						"timeout":  "30s",
					},
					Variables: map[string]client.VariableDefinition{
						"host": {
							Description: "The hostname to query.",
							Default:     "default.host.com",
						},
					},
				},
			},
		},
	}

	// Test case 1: Using runtime variable
	runtimeVars := map[string]string{"host": "runtime.host.com"}
	sc, err := cfg.GetSearchContext("test-context", []string{}, client.LogSearch{}, runtimeVars)
	require.NoError(t, err)

	assert.Equal(t, "runtime.host.com", sc.Search.Fields["hostname"])
	assert.Equal(t, "prod.server.com", sc.Search.Fields["env_host"])
	assert.Equal(t, "https://log-api/runtime.host.com", sc.Search.Options["endpoint"])

	// Test case 2: Using default variable value
	sc, err = cfg.GetSearchContext("test-context", []string{}, client.LogSearch{}, nil)
	require.NoError(t, err)

	assert.Equal(t, "default.host.com", sc.Search.Fields["hostname"], "Should fall back to default when no runtime var is provided")
	assert.Equal(t, "https://log-api/default.host.com", sc.Search.Options["endpoint"])
}

func TestGetSearchContext_InheritanceAndVariables(t *testing.T) {
	cfg := ContextConfig{
		Searches: Searches{
			"base-search": {
				Fields: ty.MS{
					"region": "${region}",
				},
				Options: ty.MI{
					"cluster": "cluster-${region}",
				},
				Variables: map[string]client.VariableDefinition{
					"region": {Default: "us-west-1"},
				},
			},
		},
		Contexts: Contexts{
			"child-context": {
				Client:        "test-client",
				SearchInherit: []string{"base-search"},
				Search: client.LogSearch{
					Fields: ty.MS{
						"service": "login-service",
					},
				},
			},
		},
	}

	// Provide 'region' at runtime
	runtimeVars := map[string]string{"region": "eu-central-1"}
	sc, err := cfg.GetSearchContext("child-context", []string{}, client.LogSearch{}, runtimeVars)
	require.NoError(t, err)

	// Check that inherited fields are resolved
	assert.Equal(t, "eu-central-1", sc.Search.Fields["region"])
	assert.Equal(t, "cluster-eu-central-1", sc.Search.Options["cluster"])
	// Check that child fields are preserved
	assert.Equal(t, "login-service", sc.Search.Fields["service"])
}

func TestGetSearchContext_EnvVariableOnly(t *testing.T) {
	t.Setenv("MY_VAR", "my_value")
	cfg := ContextConfig{
		Contexts: Contexts{
			"env-test": {
				Client: "dummy",
				Search: client.LogSearch{
					Fields: ty.MS{"key": "${MY_VAR}"},
				},
			},
		},
	}
	sc, err := cfg.GetSearchContext("env-test", []string{}, client.LogSearch{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "my_value", sc.Search.Fields["key"])
}
