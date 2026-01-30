package ty

import (
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
)

const lineBreak = "\n"
const lineRegex = "^([a-zA-Z0-9\\-]*)[:=](.*)$"

// LoadMS loads a string map from a file (KV or JSON).
// Supported separator: : , = , json map
func (ms *MS) LoadMS(path string) error {

	r := regexp.MustCompile(lineRegex)

	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }()

	value, err := io.ReadAll(file)

	if err != nil {
		return err
	}

	strValue := strings.Trim(string(value), " ")

	if strValue[0] == '{' {
		return json.Unmarshal(value, ms)
	}

	lines := strings.Split(strValue, lineBreak)

	for _, v := range lines {
		matches := r.FindAllStringSubmatch(v, len(v))
		if len(matches) == 0 || len(matches[0]) < 3 {
			continue
		}

		(*ms)[strings.Trim(matches[0][1], " ")] = strings.Trim(matches[0][2], " ")
	}

	return nil
}

// Load loads a generic map info from a JSON file.
func (mi *MI) Load(path string) error {
	err := ReadJSONFile(path, mi)
	if err != nil {
		return err
	}

	return nil
}
