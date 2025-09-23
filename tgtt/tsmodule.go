// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import "github.com/elliotchance/orderedmap/v3"

type TsModule struct {
	GoPath  string
	Imports *orderedmap.OrderedMap[string, *TsModule] // Keyed by module name
	Defs    *orderedmap.OrderedMap[string, string]
}

func (m *TsModule) Render() []byte {
	var b []byte
	b = append(b, "/* "...)
	b = append(b, m.GoPath...)
	b = append(b, " */"...)
	for path := range m.Imports.Keys() {
		b = append(b, '\n')
		b = append(b, "import * as "...)
		b = append(b, path...)
		b = append(b, ` from "./`...)
		b = append(b, path...)
		b = append(b, `";`...)
	}
	for def := range m.Defs.Values() {
		b = append(b, "\n\n"...)
		b = append(b, def...)
	}
	return b
}
