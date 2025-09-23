// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/elliotchance/orderedmap/v3"
	"github.com/hashicorp/go-set/v3"
	"golang.org/x/tools/go/packages"
)

type transpiler struct {
	primaryPkg        *packages.Package
	secondaryPkgs     []*packages.Package
	packages          map[string]*packages.Package // Keyed by package path
	modules           Package
	typeMappings      map[string]string
	includeUnexported bool
}

func (t *transpiler) mainPkgs() []*packages.Package {
	return append([]*packages.Package{t.primaryPkg}, t.secondaryPkgs...)
}

func (t *transpiler) init1() {
	t.packages = make(map[string]*packages.Package)
	stack := t.mainPkgs()
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

func (t *transpiler) init2() {
	pkgs := make([]*packages.Package, 0, len(t.packages)-1)
	for _, pkg := range t.packages {
		if pkg != t.primaryPkg {
			pkgs = append(pkgs, pkg)
		}
	}
	slices.SortFunc(pkgs, func(a, b *packages.Package) int {
		is_a_secondary := slices.Contains(t.secondaryPkgs, a)
		is_b_secondary := slices.Contains(t.secondaryPkgs, b)
		if is_a_secondary && !is_b_secondary {
			return -1
		} else if !is_a_secondary && is_b_secondary {
			return 1
		}
		if c := cmp.Compare(len(a.PkgPath), len(b.PkgPath)); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	names := set.New[string](0)
	for _, pkg := range pkgs {
		if names.Contains(pkg.Name) {
			name := make([]byte, 0, len(pkg.Name)+1+1)
			name = append(name, pkg.Name...)
			name = append(name, '_')
			for i := uint64(1); ; i++ {
				name = name[:len(pkg.Name)+1]
				name = strconv.AppendUint(name, i, 10)
				if !names.Contains(string(name)) {
					break
				}
			}
			pkg.Name = bytesToString(name)
		}
		names.Insert(pkg.Name)
	}
	t.primaryPkg.Name = "index"
}

func (t *transpiler) init3() {
	mainPkgs := t.mainPkgs()
	t.modules = make(Package, len(mainPkgs))
	for _, pkg := range mainPkgs {
		t.addModule(pkg.Name, pkg.PkgPath)
	}
}

func (t *transpiler) addModule(name, goPath string) *Module {
	mod := &Module{
		GoPath:  goPath,
		Imports: orderedmap.NewOrderedMap[string, *Module](),
		Defs:    orderedmap.NewOrderedMap[string, string](),
	}
	t.modules[name] = mod
	return mod
}

func (t *transpiler) init() {
	t.init1()
	t.init2()
	t.init3()
}
