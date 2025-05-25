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

	"github.com/fatih/structtag"
	"github.com/hashicorp/go-set/v3"
	"github.com/lindell/go-ordered-set/orderedset"
	"golang.org/x/tools/go/packages"
)

func (t *Transpiler) Transpile(names *set.Set[string]) {
	isNamesEmpty := names == nil || names.Empty()
	for _, obj := range sortedDefs(t.pkg) {
		if !isNamesEmpty && names.Contains(obj.Name()) {
			t.transpileObject(obj, t.Index())
		} else if isNamesEmpty && (obj.Exported() || t.includeUnexported) {
			t.transpileObject(obj, t.Index())
		}
	}
}

func (t *Transpiler) Index() *Module {
	return t.Modules["index"]
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
			aPos := pkg.Fset.Position(a.Pos())
			bPos := pkg.Fset.Position(b.Pos())
			if c := strings.Compare(aPos.Filename, bPos.Filename); c != 0 {
				return c
			}
			return aPos.Line - bPos.Line
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
		_, ok := obj.Type().(transpilableType)
		return ok
	default:
		return false
	}
}

type transpilableType interface {
	Obj() *types.TypeName
	Underlying() types.Type
	TypeParams() *types.TypeParamList
}

func (t *Transpiler) transpileObject(obj types.Object, mod *Module) {
	if mod.Defs.Has(obj.Name()) {
		return
	}

	// https://github.com/golang/example/tree/master/gotypes#objects
	switch obj := obj.(type) {
	case *types.Const:
		t.transpileConst(obj, mod)
	case *types.TypeName:
		t.transpileTypeDef(obj, mod)
	}
}

func (t *Transpiler) transpileConst(obj *types.Const, mod *Module) {
	mod.Defs.Set(obj.Name(), "") // prevent infinite recursion
	var def string
	switch typ := obj.Type().(type) {
	case *types.Named:
		typStr := t.transpileTypeRef(typ.Obj(), mod)
		val, ok := transpileConstVal(obj.Val(), false)
		if !ok {
			mod.Defs.Delete(obj.Name())
			return
		}
		def = fmt.Sprintf(`export const %s: %s = %s`, obj.Name(), typStr, val)
	default:
		val, ok := transpileConstVal(obj.Val(), true)
		if !ok {
			mod.Defs.Delete(obj.Name())
			return
		}
		def = fmt.Sprintf(`export const %s = %s`, obj.Name(), val)
	}
	mod.Defs.Set(obj.Name(), def)
}

func transpileConstVal(x constant.Value, allowBigint bool) (string, bool) {
	const maxSafeInt = 1<<53 - 1
	const minSafeInt = -(1<<53 - 1)
	switch x := constant.Val(x).(type) {
	case bool:
		return strconv.FormatBool(x), true
	case string:
		return strconv.Quote(x), true
	case int64:
		s := strconv.FormatInt(x, 10)
		if allowBigint && (x < minSafeInt || x > maxSafeInt) {
			s += "n" // BigInt
		}
		return s, true
	case *big.Int:
		s := x.String()
		if allowBigint && (x.Cmp(big.NewInt(minSafeInt)) == -1 || x.Cmp(big.NewInt(maxSafeInt)) == 1) {
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

func (t *Transpiler) transpileTypeDef(obj *types.TypeName, mod *Module) {
	typ, ok := obj.Type().(transpilableType)
	if !ok {
		return
	}

	mod.Defs.Set(obj.Name(), "") // prevent infinite recursion
	var def string
	{
		path := t.getPkgPath(typ.Obj())
		typStr, ok := t.typeMappings[path]
		if !ok {
			typStr = t.transpileType(typ.Underlying(), mod)
		}
		def = fmt.Sprintf(
			`export type %s%s = %s`,
			typ.Obj().Name(),
			t.transpileTypeParams(typ.TypeParams(), mod),
			typStr,
		)
	}
	mod.Defs.Set(obj.Name(), def)
}

func (t *Transpiler) transpileType(typ types.Type, mod *Module) string {
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
		return t.transpileStruct(typ, mod)
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

func (t *Transpiler) transpileBasic(typ *types.Basic, _ *Module) string {
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

func (t *Transpiler) transpilePointer(typ *types.Pointer, mod *Module) string {
	typStr := t.transpileType(typ.Elem(), mod)
	if !strings.HasSuffix(typStr, " | null") {
		typStr += " | null"
	}
	return typStr
}

func (t *Transpiler) transpileArray(typ *types.Array, mod *Module) string {
	return fmt.Sprintf(`%s[]`, t.transpileType(typ.Elem(), mod))
}

func (t *Transpiler) transpileSlice(typ *types.Slice, mod *Module) string {
	return fmt.Sprintf(`%s[]`, t.transpileType(typ.Elem(), mod))
}

func (t *Transpiler) transpileMap(typ *types.Map, mod *Module) string {
	return fmt.Sprintf(
		`{ [key in string]: %s }`, t.transpileType(typ.Elem(), mod),
	)
}

func (t *Transpiler) transpileStruct(typ *types.Struct, mod *Module) string {
	var sb strings.Builder
	s := parseStruct(typ)

	sb.WriteByte('{')
	if len(s.Fields) > 0 {
		sb.WriteByte(' ')
	}
	for _, field := range s.Fields {
		format := `%s: %s; `
		if field.Optional {
			format = `%s?: %s; `
		}
		fmt.Fprintf(
			&sb,
			format,
			strconv.Quote(field.Name),
			t.transpileType(field.Type, mod),
		)
	}
	sb.WriteByte('}')

	for _, embed := range s.Embeds {
		sb.WriteString(" & ")

		typStr := t.transpileType(embed.Type, mod)
		typStr, found := strings.CutSuffix(typStr, " | null")
		if !embed.Optional && !found {
			sb.WriteString(typStr)
		} else {
			sb.WriteString("Partial<" + typStr + ">")
		}
	}

	return sb.String()
}

type structData struct {
	Embeds, Fields []fieldData
}

type fieldData struct {
	Name     string
	Optional bool
	Type     types.Type
}

func parseStruct(typ *types.Struct) structData {
	var s structData
	for i := range typ.NumFields() {
		field := typ.Field(i)
		if !field.Exported() {
			continue
		}

		var f fieldData
		embedded := field.Embedded()
		func() {
			tags, err := structtag.Parse(typ.Tag(i))
			if tags == nil || err != nil {
				f.Name = field.Name()
				return
			}

			tag, err := tags.Get("json")
			if err != nil {
				f.Name = field.Name()
				return
			}

			f.Name = tag.Name
			f.Optional = tag.HasOption("omitempty")
			if !embedded {
				embedded = tag.HasOption("inline")
			}
		}()
		if f.Name == "-" {
			continue
		}

		f.Type = field.Type()
		if !embedded {
			s.Fields = append(s.Fields, f)
		} else {
			s.Embeds = append(s.Embeds, f)
		}
	}

	return s
}

func (t *Transpiler) transpileAlias(typ *types.Alias, mod *Module) string {
	return t.transpileTypeRef(typ.Obj(), mod) + t.transpileTypeArgs(typ.TypeArgs(), mod)
}

func (t *Transpiler) transpileNamed(typ *types.Named, mod *Module) string {
	return t.transpileTypeRef(typ.Obj(), mod) + t.transpileTypeArgs(typ.TypeArgs(), mod)
}

func (t *Transpiler) transpileInterface(typ *types.Interface, mod *Module) string {
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
	if len(unions[0]) == 0 {
		return "any"
	}

	terms := orderedset.New[string]()
	for _, termTyp := range unions[0] {
		terms.Add(t.transpileType(termTyp, mod))
	}
	return strings.Join(terms.Values(), " | ")
}

func (t *Transpiler) transpileUnion(typ *types.Union, mod *Module) string {
	terms := orderedset.New[string]()
	for term := range typ.Terms() {
		terms.Add(t.transpileType(term.Type(), mod))
	}

	if terms.Size() == 0 {
		return "any"
	}
	return strings.Join(terms.Values(), " | ")
}

func (t *Transpiler) transpileTypeParam(typ *types.TypeParam, _ *Module) string {
	return typ.Obj().Name()
}
