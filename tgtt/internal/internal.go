package internal

import "unsafe"

func BytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func Ptr[T any](v T) *T {
	return &v
}
