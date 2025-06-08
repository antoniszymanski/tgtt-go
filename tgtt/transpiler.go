/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package tgtt

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"github.com/hashicorp/go-set/v3"
	"golang.org/x/tools/go/packages"
)

type Transpiler struct {
	pkg      *packages.Package
	packages map[string]*packages.Package // Keyed by package path
	Modules  map[string]*Module           // Keyed by module name

	typeMappings      map[string]string
	includeUnexported bool
}

func (t *Transpiler) init1() {
	t.packages = make(map[string]*packages.Package)
	stack := []*packages.Package{t.pkg}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		t.packages[current.PkgPath] = current
		for _, importedPkg := range current.Imports {
			if _, ok := t.packages[importedPkg.PkgPath]; ok {
				continue
			}
			stack = append(stack, importedPkg)
		}
	}
}

func (t *Transpiler) init2() {
	pkgs := make([]*packages.Package, 0, len(t.packages)-1)
	for _, pkg := range t.packages {
		if pkg != t.pkg {
			pkgs = append(pkgs, pkg)
		}
	}
	slices.SortFunc(pkgs, func(a, b *packages.Package) int {
		if c := cmp.Compare(len(a.PkgPath), len(b.PkgPath)); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})

	names := set.New[string](0)
	for _, pkg := range pkgs {
		if !names.Contains(pkg.Name) {
			goto insertName
		}

		{
			// 20 is uint64's max string length
			name := make([]byte, 0, len(pkg.Name)+1+20)
			name = append(name, pkg.Name...)
			name = append(name, '_')
			for i := uint64(1); ; i++ {
				name = name[:len(pkg.Name)+1]
				name = strconv.AppendUint(name, i, 10)
				if !names.Contains(b2s(name)) {
					break
				}
			}
			pkg.Name = b2s(name)
		}

	insertName:
		names.Insert(pkg.Name)
	}

	t.pkg.Name = "index"
}

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func (t *Transpiler) init3() {
	t.Modules = map[string]*Module{"index": newModule(t.pkg.PkgPath)}
}

func (t *Transpiler) init() {
	t.init1()
	t.init2()
	t.init3()
}

func NewTranspiler(pattern string, opts ...TranspilerOption) (*Transpiler, error) {
	pkg, err := loadPackage(pattern)
	if err != nil {
		return nil, err
	}

	t := &Transpiler{pkg: pkg}
	for _, opt := range opts {
		opt(t)
	}
	t.init()
	return t, nil
}

type TranspilerOption func(t *Transpiler)

func TypeMappings(typeMappings map[string]string) TranspilerOption {
	return func(t *Transpiler) {
		t.typeMappings = typeMappings
	}
}

func IncludeUnexported() TranspilerOption {
	return func(t *Transpiler) {
		t.includeUnexported = true
	}
}
