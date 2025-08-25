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

func TestResolveVariablesWith(t *testing.T) {
	// 1. Setup environment variables
	t.Setenv("ENV_VAR", "env_value")
	t.Setenv("ONLY_ENV", "only_env_value")

	// 2. Define runtime variables
	runtimeVars := map[string]string{
		"RUNTIME_VAR": "runtime_value",
		"ENV_VAR":     "runtime_override", // This should take precedence over the env var
	}

	// 3. Test with MS (map[string]string)
	ms := MS{
		"runtime":     "Value is ${RUNTIME_VAR}",
		"env":         "Value is ${ONLY_ENV}",
		"override":    "Value is ${ENV_VAR}",
		"default":     "Value is ${MISSING:-default_value}",
		"unresolved":  "Value is ${NOT_FOUND}",
		"no_op":       "this is a plain string",
	}

	resolvedMs := ms.ResolveVariablesWith(runtimeVars)

	assert.Equal(t, "Value is runtime_value", resolvedMs["runtime"])
	assert.Equal(t, "Value is only_env_value", resolvedMs["env"])
	assert.Equal(t, "Value is runtime_override", resolvedMs["override"])
	assert.Equal(t, "Value is default_value", resolvedMs["default"])
	assert.Equal(t, "Value is ${NOT_FOUND}", resolvedMs["unresolved"])
	assert.Equal(t, "this is a plain string", resolvedMs["no_op"])

	// 4. Test with MI (map[string]interface{})
	mi := MI{
		"runtime":    "Value is ${RUNTIME_VAR}",
		"override":   "Value is ${ENV_VAR}",
		"default":    "Value is ${MISSING:-default_value}",
		"unresolved": "Value is ${NOT_FOUND}",
		"numeric":    123,
		"bool":       true,
	}

	resolvedMi := mi.ResolveVariablesWith(runtimeVars)

	assert.Equal(t, "Value is runtime_value", resolvedMi["runtime"])
	assert.Equal(t, "Value is runtime_override", resolvedMi["override"])
	assert.Equal(t, "Value is default_value", resolvedMi["default"])
	assert.Equal(t, "Value is ${NOT_FOUND}", resolvedMi["unresolved"])
	assert.Equal(t, 123, resolvedMi["numeric"])
	assert.Equal(t, true, resolvedMi["bool"])
}
