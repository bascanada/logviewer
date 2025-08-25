package resolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	t.Setenv("ENV_VAR", "env_value")
	t.Setenv("ONLY_ENV", "only_env_value")

	vars := map[string]string{
		"RUNTIME_VAR": "runtime_value",
		"ENV_VAR":     "runtime_override", // This should take precedence over the env var
	}

	assert.Equal(t, "runtime_value", Resolve("${RUNTIME_VAR}", vars))
	assert.Equal(t, "only_env_value", Resolve("${ONLY_ENV}", vars))
	assert.Equal(t, "runtime_override", Resolve("${ENV_VAR}", vars))
	assert.Equal(t, "default_value", Resolve("${MISSING:-default_value}", vars))
	assert.Equal(t, "${NOT_FOUND}", Resolve("${NOT_FOUND}", vars))
	assert.Equal(t, "this is a plain string", Resolve("this is a plain string", vars))
}
