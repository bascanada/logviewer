package ty

import (
	"os"
	"regexp"
)

// ResolveVars resolves shell-style variable references in the input string using the provided runtime variables and environment variables.
func ResolveVars(input string, runtimeVars map[string]string) string {
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
			return ResolveVars(matches[4], runtimeVars)
		}

		// Return the original placeholder if no value is found
		return v
	})
}

// ResolveVariables resolves variables in all values of the MS map.
func (ms MS) ResolveVariables() MS {
	return ms.ResolveVariablesWith(nil)
}

// ResolveVariablesWith resolves variables in all values of the MS map using additional runtime variables.
func (ms MS) ResolveVariablesWith(vars map[string]string) MS {
	msResolved := MS{}
	for k, v := range ms {
		msResolved[k] = ResolveVars(v, vars)
	}
	return msResolved
}

// ResolveVariables on MI resolves any string values containing shell-style
// ${VAR} or ${VAR:-default} / $VAR patterns using the same underlying logic as MS.
// Non-string values are copied unchanged.
func (mi MI) ResolveVariables() MI {
	return mi.ResolveVariablesWith(nil)
}

// ResolveVariablesWith on MI resolves variables in string values of the MI map using additional runtime variables.
func (mi MI) ResolveVariablesWith(vars map[string]string) MI {
	resolved := MI{}
	for k, v := range mi {
		switch vv := v.(type) {
		case string:
			resolved[k] = ResolveVars(vv, vars)
		default:
			resolved[k] = v
		}
	}
	return resolved
}
