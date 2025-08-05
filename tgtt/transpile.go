// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import (
	"cmp"
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

func Transpile(opts TranspileOptions) (TsPackage, error) {
	t := &transpiler{
		typeMappings:      opts.TypeMappings,
		includeUnexported: opts.IncludeUnexported,
	}
	var err error
	t.primaryPkg, err = loadPackage(opts.PrimaryPackage.Path)
	if err != nil {
		return nil, err
	}
	for _, pkgOpts := range opts.SecondaryPackages {
		pkg, err := loadPackage(pkgOpts.Path)
		if err != nil {
			return nil, err
		}
		t.secondaryPkgs = append(t.secondaryPkgs, pkg)
	}
	t.init()

	transpilePkg := func(pkg *packages.Package, opts PackageOptions) {
		isNamesEmpty := opts.Names == nil || opts.Names.Empty()
		for _, obj := range sortedDefs(pkg) {
			if !isNamesEmpty && opts.Names.Contains(obj.Name()) {
				t.transpileObject(obj, t.modules[pkg.Name])
			} else if isNamesEmpty && (obj.Exported() || t.includeUnexported) {
				t.transpileObject(obj, t.modules[pkg.Name])
			}
		}
	}
	transpilePkg(t.primaryPkg, opts.PrimaryPackage)
	for i, pkg := range t.secondaryPkgs {
		transpilePkg(pkg, opts.SecondaryPackages[i])
	}

	return t.modules, nil
}

type TranspileOptions struct {
	PrimaryPackage    PackageOptions
	SecondaryPackages []PackageOptions
	TypeMappings      map[string]string
	IncludeUnexported bool
}

type PackageOptions struct {
	Path  string           `json:"path" jsonschema:"required,minLength=1"`
	Names *set.Set[string] `json:"names"`
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
			return cmp.Compare(aPos.Line, bPos.Line)
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

func (t *transpiler) transpileObject(obj types.Object, mod *TsModule) {
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

func (t *transpiler) transpileConst(obj *types.Const, mod *TsModule) {
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

func (t *transpiler) transpileTypeDef(obj *types.TypeName, mod *TsModule) {
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

func (t *transpiler) transpileType(typ types.Type, mod *TsModule) string {
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

func (t *transpiler) transpileBasic(typ *types.Basic, _ *TsModule) string {
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

func (t *transpiler) transpilePointer(typ *types.Pointer, mod *TsModule) string {
	typStr := t.transpileType(typ.Elem(), mod)
	if !strings.HasSuffix(typStr, " | null") {
		typStr += " | null"
	}
	return typStr
}

func (t *transpiler) transpileArray(typ *types.Array, mod *TsModule) string {
	return fmt.Sprintf(`%s[]`, t.transpileType(typ.Elem(), mod))
}

func (t *transpiler) transpileSlice(typ *types.Slice, mod *TsModule) string {
	return fmt.Sprintf(`%s[]`, t.transpileType(typ.Elem(), mod))
}

func (t *transpiler) transpileMap(typ *types.Map, mod *TsModule) string {
	return fmt.Sprintf(
		`{ [key in string]: %s }`, t.transpileType(typ.Elem(), mod),
	)
}

func (t *transpiler) transpileStruct(typ *types.Struct, mod *TsModule) string {
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

func (t *transpiler) transpileAlias(typ *types.Alias, mod *TsModule) string {
	return t.transpileTypeRef(typ.Obj(), mod) + t.transpileTypeArgs(typ.TypeArgs(), mod)
}

func (t *transpiler) transpileNamed(typ *types.Named, mod *TsModule) string {
	return t.transpileTypeRef(typ.Obj(), mod) + t.transpileTypeArgs(typ.TypeArgs(), mod)
}

func (t *transpiler) transpileInterface(typ *types.Interface, mod *TsModule) string {
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

func (t *transpiler) transpileUnion(typ *types.Union, mod *TsModule) string {
	terms := orderedset.New[string]()
	for term := range typ.Terms() {
		terms.Add(t.transpileType(term.Type(), mod))
	}

	if terms.Size() == 0 {
		return "any"
	}
	return strings.Join(terms.Values(), " | ")
}

func (t *transpiler) transpileTypeParam(typ *types.TypeParam, _ *TsModule) string {
	return typ.Obj().Name()
}
