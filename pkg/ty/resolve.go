package ty

import (
	"os"
	"regexp"
)

func resolveVars(input string, runtimeVars map[string]string) string {
	re := regexp.MustCompile(`\$(\{([a-zA-Z_][a-zA-Z0-9_]*)(:-(.*))?\}|\$([a-zA-Z_][a-zA-Z0-9_]*))`)
	return re.ReplaceAllStringFunc(input, func(v string) string {
		// First, find the variable name, stripping ${} and $
		varName := re.ReplaceAllString(v, "$2$5")

		// Prioritize runtime variables
		if val, ok := runtimeVars[varName]; ok {
			return val
		}

		// Fallback to environment variables
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}

		// Use default value if provided
		// The regex now includes a capturing group for the default value.
		matches := re.FindStringSubmatch(v)
		if len(matches) > 4 && matches[4] != "" {
			// The default value is in matches[4]. It might contain other variables.
			return resolveVars(matches[4], runtimeVars)
		}

		// Return the original placeholder if no value is found
		return v
	})
}

func (ms MS) ResolveVariables() MS {
	return ms.ResolveVariablesWith(nil)
}

func (ms MS) ResolveVariablesWith(vars map[string]string) MS {
	msResolved := MS{}
	for k, v := range ms {
		msResolved[k] = resolveVars(v, vars)
	}
	return msResolved
}

// ResolveVariables on MI resolves any string values containing shell-style
// ${VAR} or ${VAR:-default} / $VAR patterns using the same underlying logic as MS.
// Non-string values are copied unchanged.
func (mi MI) ResolveVariables() MI {
	return mi.ResolveVariablesWith(nil)
}

func (mi MI) ResolveVariablesWith(vars map[string]string) MI {
	resolved := MI{}
	for k, v := range mi {
		switch vv := v.(type) {
		case string:
			resolved[k] = resolveVars(vv, vars)
		default:
			resolved[k] = v
		}
	}
	return resolved
}
