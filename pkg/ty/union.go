package ty

import (
	"log"
)

type UniSet[V string | int] map[string][]V

func (us UniSet[V]) Add(key string, v V) bool {
	if us[key] == nil {
		us[key] = make([]V, 1)
		us[key][0] = v
		return true
	}
	for _, vv := range us[key] {
		if vv == v {
			return false
		}
	}
	us[key] = append(us[key], v)
	return true
}

func AddField(k string, v interface{}, fields *UniSet[string]) {
	switch value := v.(type) {
	case string:
		fields.Add(k, value)
	case map[string]interface{}:
		for kk, vv := range value {
			recKey := k + "." + kk
			AddField(recKey, vv, fields)
		}
	default:
		log.Println("invalid type for field " + k)
	}
}
