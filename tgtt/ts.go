// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import (
	"github.com/elliotchance/orderedmap/v3"
	"golang.org/x/sync/errgroup"
)

type Package map[string]*Module // Keyed by module name

func (p Package) Index() *Module {
	return p["index"]
}

func (p Package) builtin() *Module {
	if module, ok := p["$builtin"]; ok {
		return module
	}
	module := &Module{
		GoPath:  "builtin",
		Imports: orderedmap.NewOrderedMap[string, *Module](),
		Defs:    orderedmap.NewOrderedMap[string, string](),
	}
	module.Defs.Set("comparable", "export type comparable = boolean | number | string | null | comparable[] | { [key: string]: comparable }")
	p["$builtin"] = module
	return module
}

type RenderOptions struct {
	Limit int
	Write func(moduleName string, data []byte) error
}

func (p Package) Render(opts RenderOptions) error {
	var g errgroup.Group
	if opts.Limit != 0 {
		g.SetLimit(opts.Limit)
	}
	for moduleName, mod := range p {
		g.Go(func() error {
			return opts.Write(moduleName, mod.Render())
		})
	}
	return g.Wait()
}

type Module struct {
	GoPath  string
	Imports *orderedmap.OrderedMap[string, *Module] // Keyed by module name
	Defs    *orderedmap.OrderedMap[string, string]
}

func (m *Module) Render() []byte {
	var b []byte
	b = append(b, "/* "...)
	b = append(b, m.GoPath...)
	b = append(b, " */"...)
	for moduleName := range m.Imports.Keys() {
		b = append(b, '\n')
		b = append(b, "import * as "...)
		b = append(b, moduleName...)
		b = append(b, ` from "./`...)
		b = append(b, moduleName...)
		b = append(b, `";`...)
	}
	for def := range m.Defs.Values() {
		b = append(b, "\n\n"...)
		b = append(b, def...)
	}
	return b
}
