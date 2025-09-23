// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import "github.com/fatih/structtag"

type structInfo[T any] struct {
	Fields   []fieldInfo[T]
	Embedded []T
}

type fieldInfo[T any] struct {
	Name     string
	Optional bool
	Type     T
}

func parseFieldTag[T any](s string) func(f *fieldInfo[T]) (skip bool) {
	afterParse := func(f *fieldInfo[T]) bool {
		switch f.Name {
		case "-":
			return true
		case "'-'":
			f.Name = "-"
		}
		return false
	}
	tags, err := structtag.Parse(s)
	if err != nil {
		return afterParse
	}
	tag, err := tags.Get("json")
	if err != nil {
		return afterParse
	}
	inline := tag.HasOption("inline")
	optional := tag.HasOption("omitempty") || tag.HasOption("omitzero")
	return func(f *fieldInfo[T]) bool {
		if inline {
			f.Name = ""
		} else if tag.Name != "" {
			f.Name = tag.Name
		}
		if f.Name != "" { // embedded field cannot be optional
			f.Optional = optional
		}
		return afterParse(f)
	}
}
