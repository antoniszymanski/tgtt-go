/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package tgtt

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/elliotchance/orderedmap/v3"
	"github.com/hashicorp/go-set/v3"
)

type Module struct {
	GoPath  string
	Imports *orderedmap.OrderedMap[string, *Module]
	Defs    *orderedmap.OrderedMap[string, string]
}

type MiddlewareFunc func([]byte) ([]byte, error)

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

type state struct {
	wg          sync.WaitGroup
	err         error
	middlewares []MiddlewareFunc
	mustSkip    func(key string) bool
	write       func(key string, data []byte) error
}

func (m *Module) WriteTS(outputDir string, middlewares ...MiddlewareFunc) error {
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return err
	}

	s := state{middlewares: middlewares}
	//
	writtenFiles := set.New[string](0)
	var mu sync.Mutex
	s.mustSkip = func(key string) bool {
		if s.err != nil {
			return true
		}
		mu.Lock()
		notExists := writtenFiles.Insert(key)
		mu.Unlock()
		return !notExists
	}
	//
	s.write = func(key string, data []byte) error {
		return os.WriteFile(
			filepath.Join(outputDir, key+".ts"), data, 0600,
		)
	}

	m.queueTS(&s, "index")
	s.wg.Wait()
	return s.err
}

func (m *Module) GenerateTS(middlewares ...MiddlewareFunc) (map[string][]byte, error) {
	s := state{middlewares: middlewares}
	tsFiles := make(map[string][]byte)
	var mu sync.Mutex
	//
	s.mustSkip = func(key string) bool {
		if s.err != nil {
			return true
		}
		mu.Lock()
		_, exists := tsFiles[key]
		tsFiles[key] = nil
		mu.Unlock()
		return exists
	}
	//
	s.write = func(key string, data []byte) error {
		mu.Lock()
		tsFiles[key] = data
		mu.Unlock()
		return nil
	}

	m.queueTS(&s, "index")
	s.wg.Wait()
	return tsFiles, s.err
}

func (m *Module) queueTS(state *state, key string) {
	state.wg.Add(1)
	go m.emitTS(state, key)

	for key, imported := range m.Imports.AllFromFront() {
		go imported.queueTS(state, key)
	}
}

func (m *Module) emitTS(state *state, key string) {
	defer state.wg.Done()
	if state.mustSkip(key) {
		return
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, m)
	if err != nil {
		state.err = err
		return
	}

	data := buf.Bytes()
	for _, middleware := range state.middlewares {
		data, err = middleware(data)
		if err != nil {
			state.err = err
			return
		}
	}

	state.err = state.write(key, data)
	return
}
