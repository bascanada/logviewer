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

func GetTemplateFunctionsMap() template.FuncMap {
	return template.FuncMap{
		"Format":     FormatDate,
		"MultiLine":  MultlineFields,
		"ExpandJson": ExpandJson,
	}
}
