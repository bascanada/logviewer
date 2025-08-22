package ty

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
		return v.(MS)
	}
	return MS{}
}

func (mi MI) GetBool(key string) bool {
	if v, b := mi[key]; b {
		return v.(bool)
	}
	return false
}

func MergeM[T interface{}](parent map[string]T, child map[string]T) map[string]T {
	for k, v := range child {
		parent[k] = v
	}

	return parent
}