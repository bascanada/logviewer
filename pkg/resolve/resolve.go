package resolve

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func Resolve(input string, vars map[string]string) string {
	fmt.Printf("input: %s\n", input)
	fmt.Printf("vars: %v\n", vars)
	re := regexp.MustCompile(`$(\{([a-zA-Z_][a-zA-Z0-9_]*)(:-(.*?)?)?}|$(\$([a-zA-Z_][a-zA-Z0-9_]*)))`)
	return re.ReplaceAllStringFunc(input, func(v string) string {
		fmt.Printf("v: %s\n", v)
		parts := strings.SplitN(v, ":-", 2)
		varName := strings.Trim(parts[0], "${}")
		varName = strings.Trim(varName, "$")

		if val, ok := vars[varName]; ok {
			return val
		}

		if val, ok := os.LookupEnv(varName); ok {
			return val
		}

		if len(parts) == 2 {
			return strings.TrimSuffix(parts[1], "}")
		}

		return v
	})
}