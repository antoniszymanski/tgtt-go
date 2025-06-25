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

type ModuleRenderOptions struct {
	Formatter TsFormatter
}

type TsFormatter func([]byte) ([]byte, error)

//go:embed tsmodule.go.tmpl
var tmplSource string

var tmpl = template.Must(template.New("module").Parse(tmplSource))

func (m *TsModule) Render(opts ModuleRenderOptions) ([]byte, error) {
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, m)
	if err != nil {
		return nil, err
	}

	data := buf.Bytes()
	if opts.Formatter != nil {
		data, err = opts.Formatter(data)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}
