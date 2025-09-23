// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/elliotchance/orderedmap/v3"
)

type TsModule struct {
	GoPath  string
	Imports *orderedmap.OrderedMap[string, *TsModule] // Keyed by module name
	Defs    *orderedmap.OrderedMap[string, string]
}

//go:embed tsmodule.tmpl
var tsmoduleTmplSource string

var tsmoduleTmpl = template.Must(template.New("tsmodule").Parse(tsmoduleTmplSource))

func (m *TsModule) Render() ([]byte, error) {
	var buf bytes.Buffer
	if err := tsmoduleTmpl.Execute(&buf, m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
