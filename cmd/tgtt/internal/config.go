// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package internal

import (
	"encoding/json"

	"github.com/antoniszymanski/tgtt-go/tgtt"
)

//go:generate go run ./schemagen

type Config struct {
	Schema            string                     `json:"$schema,omitzero"`
	Format            bool                       `json:"format"`
	IncludeUnexported bool                       `json:"include_unexported"`
	OutputPath        string                     `json:"output_path"`
	TypeMappings      Object[string, string]     `json:"type_mappings"`
	PrimaryPackage    tgtt.PackageOptions        `json:"primary_package"`
	SecondaryPackages Array[tgtt.PackageOptions] `json:"secondary_packages"`
}

type Array[T any] []T

func (a Array[T]) MarshalJSON() ([]byte, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]T(a))
}

type Object[K comparable, V any] map[K]V

func (o Object[K, V]) MarshalJSON() ([]byte, error) {
	if len(o) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(map[K]V(o))
}
