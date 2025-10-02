package ty

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

type Opt[T interface{}] struct {
	Value T // inner value
	Set   bool
	Valid bool
}

func OptWrap[T interface{}](value T) Opt[T] {
	return Opt[T]{
		Value: value,
		Set:   true,
		Valid: true,
	}
}

func (i *Opt[T]) Merge(or *Opt[T]) {
	if or.Set {
		i.Value = or.Value
		i.Set = or.Set
		i.Valid = or.Valid
	}
}

func (i *Opt[T]) S(v T) {
	i.Value = v
	i.Set = true
	i.Valid = true
}

func (i *Opt[T]) N() {
	i.Valid = false
}

func (i *Opt[T]) U() {
	i.Set = false
	i.Valid = false
}

func (i *Opt[T]) UnmarshalJSON(data []byte) error {
	i.Set = true

	if string(data) == "null" {
		i.Valid = false
		return nil
	}

	if err := json.Unmarshal(data, &i.Value); err != nil {
		return err
	}

	i.Valid = true

	return nil
}
// UnmarshalYAML implements yaml.Unmarshaler for Opt[T]
func (i *Opt[T]) UnmarshalYAML(value *yaml.Node) error {
	i.Set = true
	if value.Kind == yaml.ScalarNode && value.Value == "null" {
		 i.Valid = false
		 return nil
	}
	var v T
	if err := value.Decode(&v); err != nil {
		 return err
	}
	i.Value = v
	i.Valid = true
	return nil
}

// MarshalYAML implements yaml.Marshaler for Opt[T]
func (i Opt[T]) MarshalYAML() (interface{}, error) {
	if !i.Set || !i.Valid {
		 return nil, nil
	}
	return i.Value, nil
}

func (i *Opt[T]) MarshalJSON() ([]byte, error) {
	if !i.Set {
		return []byte("null"), nil
	}
	if !i.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(i.Value)
}
