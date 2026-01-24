package ty

import (
	"encoding/json"
	"os"
)

// LB is the line break constant.
const LB = "\n"

// ReadJSONFile reads and unmarshals a JSON file into object.
func ReadJSONFile(path string, object interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, object)
}

// ToJSONString converts data to a JSON string.
func ToJSONString(data any) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSONString parses a JSON string into placeholder.
func FromJSONString(data string, placeholder any) error {
	return json.Unmarshal([]byte(data), placeholder)
}
