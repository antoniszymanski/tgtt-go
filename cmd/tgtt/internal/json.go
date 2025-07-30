package internal

import (
	"bytes"
	"encoding/json"
)

func MarshalJSON(in any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(in); err != nil {
		return nil, err
	}
	out := buf.Bytes()[:buf.Len()-1] // remove a trailing newline
	return out, nil
}

type Array[T any] []T

func (a Array[T]) MarshalJSON() ([]byte, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return MarshalJSON([]T(a))
}

type Object[K comparable, V any] map[K]V

func (o Object[K, V]) MarshalJSON() ([]byte, error) {
	if len(o) == 0 {
		return []byte("{}"), nil
	}
	return MarshalJSON(map[K]V(o))
}
