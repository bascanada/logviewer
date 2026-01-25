package ty

import (
	"fmt"
	"strings"
)

// MI is a shorthand for map[string]interface{}
type MI map[string]interface{}

// MS is a shorthand for map[string]string
type MS map[string]string

// Merge merges another MI into this one.
func (mi *MI) Merge(mi2 MI) {
	// TODO: maybe support deep inspection
	for k, v := range mi2 {
		(*mi)[k] = v
	}
}

// Merge merges another MS into this one.
func (ms *MS) Merge(ms2 MS) {
	for k, v := range ms2 {
		(*ms)[k] = v
	}
}

// GetOr returns the value for the key if it exists, otherwise the default value.
func (mi MI) GetOr(key string, def interface{}) interface{} {
	if v, b := mi[key]; b {
		return v
	}
	return def
}

// GetString returns the value as a string if it exists and is a string, otherwise empty string.
func (mi MI) GetString(key string) string {
	if v, b := mi[key]; b {
		return v.(string)
	}
	return ""
}

// GetStringOk returns the value as a string if it exists and is a string, along with true.
func (mi MI) GetStringOk(key string) (string, bool) {
	v, ok := mi[key]
	if ok {
		return v.(string), ok
	}
	return "", false
}

// GetMS returns the value as a MS if it exists and is a MS, or converts map[string]interface{} to MS.
func (mi MI) GetMS(key string) MS {
	if v, b := mi[key]; b {
		switch vv := v.(type) {
		case MS:
			return vv
		case map[string]string:
			return MS(vv)
		case MI:
			res := MS{}
			for k, val := range vv {
				res[k] = fmt.Sprint(val)
			}
			return res
		case map[string]interface{}:
			res := MS{}
			for k, val := range vv {
				res[k] = fmt.Sprint(val)
			}
			return res
		default:
			return MS{}
		}
	}
	return MS{}
}

// GetBool returns the value as a bool if it exists and is a bool, otherwise false.
func (mi MI) GetBool(key string) bool {
	if v, b := mi[key]; b {
		return v.(bool)
	}
	return false
}

// GetBoolOk returns the value as a bool if it exists and can be interpreted as boolean, along with true.
func (mi MI) GetBoolOk(key string) (bool, bool) {
	v, ok := mi[key]
	if !ok {
		return false, false
	}
	switch val := v.(type) {
	case bool:
		return val, true
	case string:
		s := strings.ToLower(val)
		if s == "true" || s == "yes" || s == "1" {
			return true, true
		}
		if s == "false" || s == "no" || s == "0" {
			return false, true
		}
	}
	// Not a recognizable boolean value
	return false, false
}

// GetListOfStringsOk returns the value as a slice of strings if it exists and can be converted, along with true.
func (mi MI) GetListOfStringsOk(key string) ([]string, bool) {
	v, ok := mi[key]
	if !ok {
		return nil, false
	}

	switch vv := v.(type) {
	case []string:
		return vv, true
	case []interface{}:
		res := make([]string, len(vv))
		for i, val := range vv {
			res[i] = fmt.Sprint(val)
		}
		return res, true
	default:
		return nil, false
	}
}

// MergeM merges two maps and returns the parent map (modified).
func MergeM[T interface{}](parent map[string]T, child map[string]T) map[string]T {
	for k, v := range child {
		parent[k] = v
	}

	return parent
}
