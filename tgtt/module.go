package tgtt

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"text/template"

	"github.com/elliotchance/orderedmap/v3"
	"github.com/hashicorp/go-set/v3"
)

type Module struct {
	GoPath  string
	Imports *orderedmap.OrderedMap[string, *Module]
	Defs    *orderedmap.OrderedMap[string, string]
}

type Middleware = func([]byte) ([]byte, error)

func NewModule(goPath string) *Module {
	return &Module{
		GoPath:  goPath,
		Imports: orderedmap.NewOrderedMap[string, *Module](),
		Defs:    orderedmap.NewOrderedMap[string, string](),
	}
}

//go:embed module.go.tmpl
var tmplSource string

var tmpl = template.Must(template.New("module").Parse(tmplSource))

func (m *Module) Generate(dir string, middlewares ...Middleware) error {
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	return m.generate(dir, "index", middlewares, set.New[string](0))
}

func (m *Module) generate(
	dir, name string,
	middlewares []Middleware,
	generated *set.Set[string],
) error {
	if generated.Contains(name) {
		return nil
	}

	var buf bytes.Buffer
	var err error
	if err = tmpl.Execute(&buf, m); err != nil {
		return err
	}

	data := buf.Bytes()
	for _, m := range middlewares {
		data, err = m(data)
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(filepath.Join(dir, name+".ts"), data, 0600)
	if err != nil {
		return err
	}

	for name, i := range m.Imports.AllFromFront() {
		err = i.generate(dir, name, middlewares, generated)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Module) GenerateToMap(middlewares ...Middleware) (map[string][]byte, error) {
	fs := make(map[string][]byte)
	if err := m.generateToMap(fs, "index", middlewares...); err != nil {
		return nil, err
	}
	return fs, nil
}

func (m *Module) generateToMap(fs map[string][]byte, name string, middlewares ...Middleware) error {
	if _, ok := fs[name]; ok {
		return nil
	}

	var buf bytes.Buffer
	var err error
	if err = tmpl.Execute(&buf, m); err != nil {
		return err
	}

	data := buf.Bytes()
	for _, m := range middlewares {
		data, err = m(data)
		if err != nil {
			return err
		}
	}
	fs[name] = data

	for name, i := range m.Imports.AllFromFront() {
		err := i.generateToMap(fs, name)
		if err != nil {
			return err
		}
	}

	return nil
}
