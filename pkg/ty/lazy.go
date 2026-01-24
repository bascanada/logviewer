package ty

import "errors"

// Lazy is a function that returns a value of type T, computing it only once.
type Lazy[T interface{}] func() (*T, error)

// GetLazy returns a Lazy function that memoizes the result of the provided function.
func GetLazy[T interface{}](lazy func() (*T, error)) Lazy[T] {
	var cache *T
	return func() (*T, error) {
		if cache != nil {
			return cache, nil
		}
		cacheTmp, err := lazy()
		if err != nil {
			return cache, err
		}
		cache = cacheTmp
		return cache, nil
	}
}

// LazyMap is a map of strings to Lazy values.
type LazyMap[K string, V interface{}] map[K]Lazy[V]

// Get retrieves the value associated with key, computing it if necessary.
func (lm LazyMap[K, V]) Get(key K) (*V, error) {
	val, ok := lm[key]
	if !ok {
		return nil, errors.New("not found " + string(key))
	}
	return val()
}
