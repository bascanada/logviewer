package ty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type TestStruct struct {
	OptionalString Opt[string] `yaml:"optionalString,omitempty"`
	OptionalInt    Opt[int]    `yaml:"optionalInt,omitempty"`
}

func TestOpt_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected TestStruct
	}{
		{
			name: "String and Int present",
			yamlData: `optionalString: "hello"
optionalInt: 123`,
			expected: TestStruct{
				OptionalString: Opt[string]{Value: "hello", Set: true, Valid: true},
				OptionalInt:    Opt[int]{Value: 123, Set: true, Valid: true},
			},
		},
		{
			name:     "String is omitted, Int is present",
			yamlData: `optionalInt: 123`,
			expected: TestStruct{
				OptionalString: Opt[string]{Set: false, Valid: false},
				OptionalInt:    Opt[int]{Value: 123, Set: true, Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestStruct
			err := yaml.Unmarshal([]byte(tt.yamlData), &result)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOpt_MarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    TestStruct
		expected string
	}{
		{
			name: "String and Int present",
			input: TestStruct{
				OptionalString: Opt[string]{Value: "hello", Set: true, Valid: true},
				OptionalInt:    Opt[int]{Value: 123, Set: true, Valid: true},
			},
			expected: `optionalString: hello
optionalInt: 123
`,
		},
		{
			name: "String is null, Int is present",
			input: TestStruct{
				OptionalString: Opt[string]{Set: true, Valid: false},
				OptionalInt:    Opt[int]{Value: 123, Set: true, Valid: true},
			},
			expected: `optionalString: null
optionalInt: 123
`,
		},
		{
			name: "String is omitted, Int is present",
			input: TestStruct{
				OptionalString: Opt[string]{Set: false, Valid: false},
				OptionalInt:    Opt[int]{Value: 123, Set: true, Valid: true},
			},
			expected: `optionalInt: 123
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlData, err := yaml.Marshal(&tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(yamlData))
		})
	}
}
