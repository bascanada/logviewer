package printer

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"text/template"
	"time"

	"github.com/TylerBrock/colorjson"
	"github.com/bascanada/logviewer/pkg/ty"
)

const (
	regexJsonExtraction = "{(?:[^{}]|(?P<recurse>{[^{}]*}))*}"
)

func FormatDate(layout string, t time.Time) string {
	return t.Format(layout)
}

func MultlineFields(values ty.MI) string {
	str := ""

	for k, v := range values {
		switch value := v.(type) {
		case string:
			str += fmt.Sprintf("\n * %s=%s", k, value)
		default:
			continue
		}
	}

	return str
}

func ExpandJson(value string) string {
	reg := regexp.MustCompile(regexJsonExtraction)
	f := colorjson.NewFormatter()
	f.Indent = 2
	str := ""
	for _, jsonStr := range reg.FindStringSubmatch(value) {
		var obj map[string]interface{}

		json.Unmarshal([]byte(jsonStr), &obj)
		s, err := f.Marshal(obj)
		if err != nil {
			log.Println("failed to unmarshal json " + jsonStr)
			return ""
		}
		str += "\n" + string(s)
	}
	return str
}

// GetField provides case-insensitive field access for templates.
// Usage in template: {{Field . "level"}} or {{Field . "thread"}}
func GetField(fields ty.MI, key string) interface{} {
	// Try exact match first
	if val, ok := fields[key]; ok {
		return val
	}
	// Try capitalized version (common for struct fields)
	if len(key) > 0 {
		capKey := string(key[0]-32) + key[1:]
		if val, ok := fields[capKey]; ok {
			return val
		}
	}
	return ""
}

func GetTemplateFunctionsMap() template.FuncMap {
	return template.FuncMap{
		"Format":     FormatDate,
		"MultiLine":  MultlineFields,
		"ExpandJson": ExpandJson,
		"Field":      GetField,
	}
}
