package ty

import (
	"os"
	"regexp"
	"strings"
)

func resolveEnvVarsWithDefault(input string) string {
	re := regexp.MustCompile(`\$(\{([a-zA-Z_][a-zA-Z0-9_]*)(:-(.*?)?)?\}|\$([a-zA-Z_][a-zA-Z0-9_]*))`)
	return re.ReplaceAllStringFunc(input, func(v string) string {
		parts := strings.SplitN(v, ":-", 2)
		varName := strings.Trim(parts[0], "${}")
		varName = strings.Trim(varName, "$")

		if val, ok := os.LookupEnv(varName); ok {
			return val
		}

		if len(parts) == 2 {
			return strings.TrimSuffix(parts[1], "}")
		}

		return v
	})
}

func (ms MS) ResolveVariables() MS {
	msResolved := MS{}

	for k, v := range ms {
		msResolved[k] = resolveEnvVarsWithDefault(v)
	}

	return msResolved
}

// ResolveVariables on MI resolves any string values containing shell-style
// ${VAR} or ${VAR:-default} / $VAR patterns using the same underlying logic as MS.
// Non-string values are copied unchanged.
func (mi MI) ResolveVariables() MI {
	resolved := MI{}
	for k, v := range mi {
		switch vv := v.(type) {
		case string:
			resolved[k] = resolveEnvVarsWithDefault(vv)
		default:
			resolved[k] = v
		}
	}
	return resolved
}
