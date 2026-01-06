package printer

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/TylerBrock/colorjson"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/fatih/color"
)

const (
	regexJsonExtraction = "{(?:[^{}]|(?P<recurse>{[^{}]*}))*}"
)

func FormatDate(layout string, t time.Time) string {
	return t.Format(layout)
}

// FormatTimestamp formats a timestamp in local time, returning "N/A" for zero-value timestamps.
// This is useful for aggregated results (stats, timechart) where timestamps may be unknown.
// Converting to local time ensures the displayed time matches what users can type in --from/--to.
// Usage in template: {{FormatTimestamp .Timestamp "15:04:05"}}
func FormatTimestamp(t time.Time, layout string) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Local().Format(layout)
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

func KV(values ty.MI) string {
	items := make([]string, 0, len(values))
	for k, v := range values {
		items = append(items, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(items, " ")
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

// Trim removes leading and trailing whitespace from a string.
// Usage in template: {{Trim .Message}} or {{.Message | Trim}}
func Trim(s string) string {
	return strings.TrimSpace(s)
}

// ColorLevel applies color based on log level.
// Usage in template: {{ColorLevel .Level}}
// Color mapping: ERROR/FATAL/CRITICAL=red, WARN/WARNING=yellow, INFO=cyan, DEBUG=blue, TRACE=dim
func ColorLevel(level string) string {
	if !IsColorEnabled() {
		return level
	}

	levelUpper := strings.ToUpper(strings.TrimSpace(level))

	switch levelUpper {
	case "ERROR", "FATAL", "CRITICAL":
		return color.RedString(level)
	case "WARN", "WARNING":
		return color.YellowString(level)
	case "INFO":
		return color.CyanString(level)
	case "DEBUG":
		return color.BlueString(level)
	case "TRACE":
		return color.New(color.FgHiBlack).Sprint(level)
	default:
		return level
	}
}

// ColorTimestamp colors timestamp in dim gray.
// Usage in template: {{ColorTimestamp (FormatTimestamp .Timestamp "15:04:05")}}
func ColorTimestamp(timestamp string) string {
	if !IsColorEnabled() {
		return timestamp
	}
	return color.New(color.FgHiBlack).Sprint(timestamp)
}

// ColorContext colors context ID in magenta.
// Usage in template: {{ColorContext .ContextID}}
func ColorContext(contextID string) string {
	if !IsColorEnabled() {
		return contextID
	}
	return color.MagentaString(contextID)
}

// ColorString applies a named color to text.
// Usage in template: {{ColorString "red" "ERROR"}} or {{ColorString "green" .Message}}
// Available colors: red, green, yellow, blue, magenta, cyan, white, black, dim/gray/grey
func ColorString(colorName, text string) string {
	if !IsColorEnabled() {
		return text
	}

	switch strings.ToLower(colorName) {
	case "red":
		return color.RedString(text)
	case "green":
		return color.GreenString(text)
	case "yellow":
		return color.YellowString(text)
	case "blue":
		return color.BlueString(text)
	case "magenta":
		return color.MagentaString(text)
	case "cyan":
		return color.CyanString(text)
	case "white":
		return color.WhiteString(text)
	case "black":
		return color.BlackString(text)
	case "dim", "gray", "grey":
		return color.New(color.FgHiBlack).Sprint(text)
	default:
		return text
	}
}

// Bold makes text bold.
// Usage in template: {{Bold "Important Message"}} or {{Bold .Level}}
func Bold(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return color.New(color.Bold).Sprint(text)
}

func GetTemplateFunctionsMap() template.FuncMap {
	return template.FuncMap{
		"Format":          FormatDate,
		"FormatTimestamp": FormatTimestamp,
		"MultiLine":       MultlineFields,
		"ExpandJson":      ExpandJson,
		"Field":           GetField,
		"KV":              KV,
		"Trim":            Trim,
		// Color functions
		"ColorLevel":     ColorLevel,
		"ColorTimestamp": ColorTimestamp,
		"ColorContext":   ColorContext,
		"ColorString":    ColorString,
		"Bold":           Bold,
	}
}
