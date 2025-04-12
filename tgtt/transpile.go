/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package tgtt

import (
	"fmt"
	"go/constant"
	"go/types"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/antoniszymanski/bimap-go"
	"github.com/hashicorp/go-set/v3"
	"github.com/lindell/go-ordered-set/orderedset"
	"golang.org/x/tools/go/packages"
)

type transpiler struct {
	pkg          *packages.Package
	pkgNames     bimap.BiMap[string, string]
	Modules      map[string]*Module
	TypeMappings map[string]string
}

func NewTranspiler(pkg *packages.Package) *transpiler {
	return &transpiler{
		pkg:      pkg,
		pkgNames: bimap.New[string, string](0),
		Modules: map[string]*Module{
			"index": NewModule(pkg.PkgPath),
		},
		TypeMappings: make(map[string]string),
	}
}

func (t *transpiler) Index() *Module {
	return t.Modules["index"]
}

func (t *transpiler) Transpile(names *set.Set[string]) {
	for _, obj := range sortedDefs(t.pkg) {
		if names == nil || names.Size() == 0 || names.Contains(obj.Name()) {
			t.transpileObject(obj, t.Modules["index"])
		}
	}
}

func sortedDefs(pkg *packages.Package) []types.Object {
	var defs []types.Object
	for _, obj := range pkg.TypesInfo.Defs {
		if isTranspilable(obj) {
			defs = append(defs, obj)
		}
	}
	slices.SortFunc(
		defs,
		func(a, b types.Object) int {
			return strings.Compare(a.Name(), b.Name())
		},
	)
	return defs
}

func isTranspilable(obj types.Object) bool {
	if obj == nil {
		return false
	}
	// https://github.com/golang/example/tree/master/gotypes#objects
	switch obj := obj.(type) {
	case *types.Const:
		return constant.Val(obj.Val()) != nil
	case *types.TypeName:
		_, ok := obj.Type().(topLevel)
		return ok
	default:
		return false
	}
}

func (t *transpiler) transpileObject(obj types.Object, mod *Module) {
	if mod.Defs.Has(obj.Name()) {
		return
	}

	// https://github.com/golang/example/tree/master/gotypes#objects
	switch obj := obj.(type) {
	case *types.Const:
		t.transpileConst(obj, mod)
	case *types.TypeName:
		t.transpileTypeName(obj, mod)
	}
}

func (t *transpiler) transpileConst(obj *types.Const, mod *Module) {
	mod.Defs.Set(obj.Name(), "") // prevent infinite recursion
	var def string
	switch typ := obj.Type().(type) {
	case *types.Named:
		tobj := typ.Obj()
		pkg := tobj.Pkg()
		typStr := t.include(mod, pkg, tobj.Name())

		val, ok := transpileConstVal(obj.Val())
		if !ok {
			mod.Defs.Delete(obj.Name())
			return
		}
		def = fmt.Sprintf(`const %s: %s = %s`, obj.Name(), typStr, val)
	default:
		val, ok := transpileConstVal(obj.Val())
		if !ok {
			mod.Defs.Delete(obj.Name())
			return
		}
		def = fmt.Sprintf(`const %s = %s`, obj.Name(), val)
	}
	mod.Defs.Set(obj.Name(), def)
}

func transpileConstVal(x constant.Value) (string, bool) {
	const maxSafeInt = 1<<53 - 1
	const minSafeInt = -(1<<53 - 1)
	switch x := constant.Val(x).(type) {
	case bool:
		return strconv.FormatBool(x), true
	case string:
		return strconv.Quote(x), true
	case int64:
		s := strconv.FormatInt(x, 10)
		if x < minSafeInt || x > maxSafeInt {
			s += "n" // BigInt
		}
		return s, true
	case *big.Int:
		s := x.String()
		if x.Cmp(big.NewInt(minSafeInt)) == -1 ||
			x.Cmp(big.NewInt(maxSafeInt)) == 1 {
			s += "n" // BigInt
		}
		return s, true
	case *big.Rat:
		f, _ := x.Float64()
		return strconv.FormatFloat(f, 'g', -1, 64), true
	case *big.Float:
		f, _ := x.Float64()
		return strconv.FormatFloat(f, 'g', -1, 64), true
	default:
		return "", false
	}
}

func (t *transpiler) transpileTypeName(obj *types.TypeName, mod *Module) {
	typ, ok := obj.Type().(topLevel)
	if !ok {
		return
	}

	mod.Defs.Set(obj.Name(), "") // prevent infinite recursion
	var def string
	switch typ.Underlying().(type) {
	case *types.Struct:
		def = t.transpileStruct(typ, mod)
	default:
		def = t.transpileToplevel(typ, mod)
	}
	mod.Defs.Set(obj.Name(), def)
}

func (t *transpiler) transpileType(typ types.Type, mod *Module) string {
	// https://github.com/golang/example/tree/master/gotypes#types
	switch typ := typ.(type) {
	case *types.Basic:
		return t.transpileBasic(typ, mod)
	case *types.Pointer:
		return t.transpilePointer(typ, mod)
	case *types.Array:
		return t.transpileArray(typ, mod)
	case *types.Slice:
		return t.transpileSlice(typ, mod)
	case *types.Map:
		return t.transpileMap(typ, mod)
	case *types.Struct:
		return t.transpileStructBody(parseStruct(typ), mod)
	case *types.Alias:
		return t.transpileAlias(typ, mod)
	case *types.Named:
		return t.transpileNamed(typ, mod)
	case *types.Interface:
		return t.transpileInterface(typ, mod)
	case *types.Union:
		return t.transpileUnion(typ, mod)
	case *types.TypeParam:
		return t.transpileTypeParam(typ, mod)
	default:
		return "any"
	}
}

func (t *transpiler) transpileBasic(typ *types.Basic, _ *Module) string {
	switch typ.Kind() {
	case types.Bool:
		return "boolean"
	case types.Int:
		return "number /* int */"
	case types.Int8:
		return "number /* int8 */"
	case types.Int16:
		return "number /* int16 */"
	case types.Int32:
		return "number /* int32 */"
	case types.Int64:
		return "number /* int64 */"
	case types.Uint:
		return "number /* uint */"
	case types.Uint8:
		return "number /* uint8 */"
	case types.Uint16:
		return "number /* uint16 */"
	case types.Uint32:
		return "number /* uint32 */"
	case types.Uint64:
		return "number /* uint64 */"
	case types.Uintptr:
		return "number /* uintptr */"
	case types.Float32:
		return "number /* float32 */"
	case types.Float64:
		return "number /* float64 */"
	case types.String:
		return "string"
	default:
		return "any"
	}
}

func (t *transpiler) transpilePointer(typ *types.Pointer, mod *Module) string {
	typStr := t.transpileType(typ.Elem(), mod)
	if !strings.HasSuffix(typStr, " | null") {
		typStr += " | null"
	}
	return typStr
}

func (t *transpiler) transpileArray(typ *types.Array, mod *Module) string {
	return fmt.Sprintf(`%s[]`, t.transpileType(typ.Elem(), mod))
}

func (t *transpiler) transpileSlice(typ *types.Slice, mod *Module) string {
	return fmt.Sprintf(`%s[]`, t.transpileType(typ.Elem(), mod))
}

func (t *transpiler) transpileMap(typ *types.Map, mod *Module) string {
	return fmt.Sprintf(
		`{ [key in string]: %s }`, t.transpileType(typ.Elem(), mod),
	)
}

func (t *transpiler) transpileAlias(typ *types.Alias, mod *Module) string {
	return t.transpileTypeArgs(typ, mod)
}

func (t *transpiler) transpileNamed(typ *types.Named, mod *Module) string {
	if typ.Obj().Pkg() == nil && typ.Obj().Name() == "comparable" {
		return "string | number /* comparable */"
	}
	return t.transpileTypeArgs(typ, mod)
}

func (t *transpiler) transpileInterface(typ *types.Interface, mod *Module) string {
	intersect := func(a, b []types.Type) []types.Type {
		var dest []types.Type
		for _, x := range a {
			for _, y := range b {
				if types.Identical(x, y) {
					dest = append(dest, x)
				}
			}
		}
		return dest
	}

	var unions [][]types.Type
	for e := range typ.EmbeddedTypes() {
		var terms []types.Type
		if u, ok := e.(*types.Union); ok {
			for term := range u.Terms() {
				terms = append(terms, term.Type())
			}
			terms = slices.CompactFunc(terms, types.Identical)
		} else {
			terms = []types.Type{e}
		}
		unions = append(unions, terms)
	}
	if len(unions) == 0 {
		return "any"
	}

	for _, y := range unions[1:] {
		unions[0] = intersect(unions[0], y)
	}

	terms := orderedset.New[string]()
	for _, termTyp := range unions[0] {
		terms.Add(t.transpileType(termTyp, mod))
	}
	return strings.Join(terms.Values(), " | ")
}

func (t *transpiler) transpileUnion(typ *types.Union, mod *Module) string {
	terms := orderedset.New[string]()
	for term := range typ.Terms() {
		terms.Add(t.transpileType(term.Type(), mod))
	}

	if terms.Size() == 0 {
		return "any"
	}
	return strings.Join(terms.Values(), " | ")
}

func (t *transpiler) transpileTypeParam(typ *types.TypeParam, _ *Module) string {
	return typ.Obj().Name()
}
