package ty

import (
	"fmt"
	"strings"
)

type MI map[string]interface{}
type MS map[string]string

func (mi *MI) Merge(mi2 MI) {
	// TODO: maybe support deep inspection
	for k, v := range mi2 {
		(*mi)[k] = v
	}
}

func (ms *MS) Merge(ms2 MS) {
	for k, v := range ms2 {
		(*ms)[k] = v
	}
}

func (mi MI) GetOr(key string, def interface{}) interface{} {
	if v, b := mi[key]; b {
		return v
	}
	return def
}

func (mi MI) GetString(key string) string {
	if v, b := mi[key]; b {
		return v.(string)
	}
	return ""
}

func (mi MI) GetStringOk(key string) (string, bool) {
	v, ok := mi[key]
	if ok {
		return v.(string), ok
	}
	return "", false
}

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

func (mi MI) GetBool(key string) bool {
	if v, b := mi[key]; b {
		return v.(bool)
	}
	return false
}

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

func MergeM[T interface{}](parent map[string]T, child map[string]T) map[string]T {
	for k, v := range child {
		parent[k] = v
	}

	return parent
}
