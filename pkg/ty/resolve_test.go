package ty

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveString(t *testing.T) {

	t.Setenv("LOVE", "love")
	t.Setenv("BLIND", "visible")

	ms := MS{
		"test": "${LOVE}-is-${BLIND}",
	}

	resolvedMs := ms.ResolveVariables()

	assert.Equal(t, "love-is-visible", resolvedMs["test"], "failed to correctlty resolved varialbes")

}

func TestResolveNoEnv(t *testing.T) {

	ms := MS{
		"test": "${LOVE}-is-${BLIND}",
	}

	resolvedMs := ms.ResolveVariables()

	assert.Equal(t, "${LOVE}-is-${BLIND}", resolvedMs["test"], "failed to correctlty resolved varialbes")

}

func TestResolveStringDefault(t *testing.T) {

	t.Setenv("LOVE", "love")

	ms := MS{
		"test": "${LOVE}-is-${BLIND:-blind}",
	}

	resolvedMs := ms.ResolveVariables()

	assert.Equal(t, "love-is-blind", resolvedMs["test"], "failed to correctlty resolved varialbes")

}

func TestMI_ResolveVariablesWith(t *testing.T) {
	// Set an environment variable to ensure it's used as a fallback
	t.Setenv("ENV_VAR", "env_value")

	mi := MI{
		"from_runtime": "${RUNTIME_VAR}",
		"from_env":     "${ENV_VAR}",
		"with_default": "${UNDEFINED:-default_value}",
		"no_change":    "just a string",
		"nested":       "${RUNTIME_VAR:-${ENV_VAR}}",
	}

	runtimeVars := map[string]string{
		"RUNTIME_VAR": "runtime_value",
	}

	resolvedMI := mi.ResolveVariablesWith(runtimeVars)

	expected := MI{
		"from_runtime": "runtime_value",
		"from_env":     "env_value",
		"with_default": "default_value",
		"no_change":    "just a string",
		"nested":       "runtime_value",
	}

	assert.Equal(t, expected, resolvedMI)
}

func TestMS_ResolveVariablesWith_Priority(t *testing.T) {
	// Set an environment variable that will be overridden
	t.Setenv("SESSION_ID", "env_session_id")

	ms := MS{
		"query": "SELECT * FROM logs WHERE session_id = '${SESSION_ID}'",
	}

	runtimeVars := map[string]string{
		"SESSION_ID": "runtime_session_id",
	}

	resolvedMS := ms.ResolveVariablesWith(runtimeVars)

	expected := MS{
		"query": "SELECT * FROM logs WHERE session_id = 'runtime_session_id'",
	}

	assert.Equal(t, expected, resolvedMS)
}
