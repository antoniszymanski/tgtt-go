// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import (
	"bytes"
	"cmp"
	"go/constant"
	"go/types"
	"hash/maphash"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/antoniszymanski/collections-go/set"
	"github.com/antoniszymanski/loadpackage-go"
	"golang.org/x/tools/go/packages"
)

func Transpile(opts TranspileOptions) (Package, error) {
	t := &transpiler{
		typeMappings:      opts.TypeMappings,
		includeUnexported: opts.IncludeUnexported,
		fallbackType:      opts.FallbackType,
	}
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedTypesInfo,
	}
	var err error
	t.primaryPkg, err = loadpackage.Load("pattern="+opts.PrimaryPackage.Path, cfg)
	if err != nil {
		return nil, err
	}
	for _, pkgOpts := range opts.SecondaryPackages {
		pkg, err := loadpackage.Load("pattern="+pkgOpts.Path, cfg)
		if err != nil {
			return nil, err
		}
		t.secondaryPkgs = append(t.secondaryPkgs, pkg)
	}
	t.init()

	transpile := func(pkg *packages.Package, names set.Set[string]) {
		for _, obj := range sortedDefs(pkg) {
			if !names.Empty() && names.Contains(obj.Name()) {
				t.transpileObject(obj, t.modules[pkg.Name])
			} else if names.Empty() && (obj.Exported() || t.includeUnexported) {
				t.transpileObject(obj, t.modules[pkg.Name])
			}
		}
	}
	transpile(t.primaryPkg, opts.PrimaryPackage.Names)
	for i, pkg := range t.secondaryPkgs {
		transpile(pkg, opts.SecondaryPackages[i].Names)
	}

	if index := t.modules.Index(); index != nil {
		for _, name := range opts.Include {
			if name == "" || index.Imports.Has(name) {
				continue
			}
			if module := t.modules[name]; module != nil {
				index.Imports.Set(name, module)
			}
		}
	}
	return t.modules, nil
}

type TranspileOptions struct {
	PrimaryPackage    PackageOptions
	Include           []string
	SecondaryPackages []PackageOptions
	TypeMappings      map[string]string
	IncludeUnexported bool
	FallbackType      string
}

type PackageOptions struct {
	Path  string          `json:"path" jsonschema:"required,minLength=1"`
	Names set.Set[string] `json:"names"`
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

func (t *transpiler) transpileObject(obj types.Object, mod *Module) {
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

func (t *transpiler) transpileConst(obj *types.Const, mod *Module) {
	mod.Defs.Set(obj.Name(), "") // prevent infinite recursion
	def := append([]byte(nil), "export const "...)
	def = append(def, obj.Name()...)
	var ok bool
	switch typ := obj.Type().(type) {
	case *types.Named:
		def = append(def, ": "...)
		def = t.transpileTypeRef(def, typ.Obj(), mod)
		def = append(def, " = "...)
		def, ok = transpileConstVal(def, obj.Val(), false)
	default:
		def = append(def, " = "...)
		def, ok = transpileConstVal(def, obj.Val(), true)
	}
	if !ok {
		mod.Defs.Delete(obj.Name())
		return
	}
	mod.Defs.Set(obj.Name(), bytesToString(def))
}

func transpileConstVal(dst []byte, x constant.Value, allowBigint bool) ([]byte, bool) {
	const maxSafeInt = 1<<53 - 1
	const minSafeInt = -(1<<53 - 1)
	switch x := constant.Val(x).(type) {
	case bool:
		return strconv.AppendBool(dst, x), true
	case string:
		return strconv.AppendQuote(dst, x), true
	case int64:
		dst = strconv.AppendInt(dst, x, 10)
		if allowBigint && (x < minSafeInt || x > maxSafeInt) {
			dst = append(dst, 'n') // BigInt
		}
		return dst, true
	case *big.Int:
		dst = x.Append(dst, 10)
		if allowBigint && (x.Cmp(big.NewInt(minSafeInt)) == -1 || x.Cmp(big.NewInt(maxSafeInt)) == 1) {
			dst = append(dst, 'n') // BigInt
		}
		return dst, true
	case *big.Rat:
		f, _ := x.Float64()
		return strconv.AppendFloat(dst, f, 'g', -1, 64), true
	case *big.Float:
		f, _ := x.Float64()
		return strconv.AppendFloat(dst, f, 'g', -1, 64), true
	default:
		return nil, false
	}
}

func (t *transpiler) transpileTypeDef(obj *types.TypeName, mod *Module) {
	typ, ok := obj.Type().(transpilableType)
	if !ok {
		return
	}
	mod.Defs.Set(obj.Name(), "") // prevent infinite recursion
	def := append([]byte(nil), "export type "...)
	def = append(def, typ.Obj().Name()...)
	def = t.transpileTypeParams(def, typ.TypeParams(), mod)
	def = append(def, " = "...)
	qualifiedName := t.qualifiedName(typ.Obj())
	if x, ok := t.typeMappings[qualifiedName]; ok {
		def = append(def, x...)
	} else {
		def = t.transpileType(def, typ.Underlying(), mod)
	}
	mod.Defs.Set(obj.Name(), bytesToString(def))
}

func (t *transpiler) transpileType(dst []byte, typ types.Type, mod *Module) []byte {
	// https://github.com/golang/example/tree/master/gotypes#types
	switch typ := typ.(type) {
	case *types.Basic:
		return t.transpileBasic(dst, typ, mod)
	case *types.Pointer:
		return t.transpilePointer(dst, typ, mod)
	case *types.Array:
		return t.transpileArray(dst, typ, mod)
	case *types.Slice:
		return t.transpileSlice(dst, typ, mod)
	case *types.Map:
		return t.transpileMap(dst, typ, mod)
	case *types.Struct:
		return t.transpileStruct(dst, typ, mod)
	case *types.Alias:
		return t.transpileAlias(dst, typ, mod)
	case *types.Named:
		return t.transpileNamed(dst, typ, mod)
	case *types.Interface:
		return t.transpileInterface(dst, typ, mod)
	case *types.Union:
		return t.transpileUnion(dst, typ, mod)
	case *types.TypeParam:
		return t.transpileTypeParam(dst, typ, mod)
	default:
		return append(dst, t.fallbackType...)
	}
}

func (t *transpiler) transpileBasic(dst []byte, typ *types.Basic, _ *Module) []byte {
	if x, ok := t.typeMappings["_."+typ.Name()]; ok {
		return append(dst, x...)
	}
	switch typ.Kind() {
	case types.Bool:
		return append(dst, "boolean"...)
	case types.Int:
		return append(dst, "number /* int */"...)
	case types.Int8:
		return append(dst, "number /* int8 */"...)
	case types.Int16:
		return append(dst, "number /* int16 */"...)
	case types.Int32:
		return append(dst, "number /* int32 */"...)
	case types.Int64:
		return append(dst, "number /* int64 */"...)
	case types.Uint:
		return append(dst, "number /* uint */"...)
	case types.Uint8:
		return append(dst, "number /* uint8 */"...)
	case types.Uint16:
		return append(dst, "number /* uint16 */"...)
	case types.Uint32:
		return append(dst, "number /* uint32 */"...)
	case types.Uint64:
		return append(dst, "number /* uint64 */"...)
	case types.Uintptr:
		return append(dst, "number /* uintptr */"...)
	case types.Float32:
		return append(dst, "number /* float32 */"...)
	case types.Float64:
		return append(dst, "number /* float64 */"...)
	case types.String:
		return append(dst, "string"...)
	default:
		return append(dst, t.fallbackType...)
	}
}

func (t *transpiler) transpilePointer(dst []byte, typ *types.Pointer, mod *Module) []byte {
	dst = t.transpileType(dst, typ.Elem(), mod)
	if !bytes.HasSuffix(dst, []byte(" | null")) {
		dst = append(dst, " | null"...)
	}
	return dst
}

func (t *transpiler) transpileArray(dst []byte, typ *types.Array, mod *Module) []byte {
	dst = t.transpileType(dst, typ.Elem(), mod)
	dst = append(dst, "[]"...)
	return dst
}

func (t *transpiler) transpileSlice(dst []byte, typ *types.Slice, mod *Module) []byte {
	dst = t.transpileType(dst, typ.Elem(), mod)
	dst = append(dst, "[]"...)
	return dst
}

func (t *transpiler) transpileMap(dst []byte, typ *types.Map, mod *Module) []byte {
	dst = append(dst, "{ [key in string]: "...)
	dst = t.transpileType(dst, typ.Elem(), mod)
	dst = append(dst, " }"...)
	return dst
}

func (t *transpiler) transpileStruct(dst []byte, typ *types.Struct, mod *Module) []byte {
	s := parseStruct(typ)
	dst = append(dst, '{')
	if len(s.Fields) > 0 {
		dst = append(dst, ' ')
	}
	for i, field := range s.Fields {
		dst = strconv.AppendQuote(dst, field.Name)
		if field.Optional {
			dst = append(dst, '?')
		}
		dst = append(dst, ": "...)
		dst = t.transpileType(dst, field.Type, mod)
		if i < len(s.Fields)-1 {
			dst = append(dst, ';')
		}
		dst = append(dst, ' ')
	}
	dst = append(dst, '}')
	for _, embedded := range s.Embedded {
		dst = append(dst, " & "...)
		i := len(dst)
		dst = t.transpileType(dst, embedded, mod)
		var found bool
		dst, found = bytes.CutSuffix(dst, []byte(" | null"))
		if found {
			dst = slices.Insert(dst, i, []byte("Partial<")...)
			dst = append(dst, '>')
		}
	}
	return dst
}

func parseStruct(typ *types.Struct) structInfo[types.Type] {
	var s structInfo[types.Type]
	for i := range typ.NumFields() {
		field := typ.Field(i)
		if !field.Exported() {
			continue
		}
		f := fieldInfo[types.Type]{Name: field.Name(), Type: field.Type()}
		if field.Embedded() {
			f.Name = ""
		}
		if parseFieldTag[types.Type](typ.Tag(i))(&f) {
			continue
		}
		if f.Name == "" {
			s.Embedded = append(s.Embedded, f.Type)
		} else {
			s.Fields = append(s.Fields, f)
		}
	}
	return s
}

func (t *transpiler) transpileAlias(dst []byte, typ *types.Alias, mod *Module) []byte {
	dst = t.transpileTypeRef(dst, typ.Obj(), mod)
	dst = t.transpileTypeArgs(dst, typ.TypeArgs(), mod)
	return dst
}

func (t *transpiler) transpileNamed(dst []byte, typ *types.Named, mod *Module) []byte {
	dst = t.transpileTypeRef(dst, typ.Obj(), mod)
	dst = t.transpileTypeArgs(dst, typ.TypeArgs(), mod)
	return dst
}

func (t *transpiler) transpileInterface(dst []byte, typ *types.Interface, mod *Module) []byte {
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
		return append(dst, t.fallbackType...)
	}
	for _, y := range unions[1:] {
		unions[0] = intersect(unions[0], y)
	}
	if len(unions[0]) == 0 {
		return append(dst, t.fallbackType...)
	}
	s := set.New[uint64](0)
	seed := maphash.MakeSeed()
	insert := func(b []byte) (modified bool) {
		hash := maphash.Bytes(seed, b)
		if s.Contains(hash) {
			return false
		}
		s.Insert(hash)
		return true
	}
	for _, typ := range unions[0] {
		i := len(dst)
		dst = t.transpileType(dst, typ, mod)
		if !insert(dst[i:]) {
			dst = dst[:i]
		} else {
			dst = append(dst, " | "...)
		}
	}
	dst = bytes.TrimSuffix(dst, []byte(" | "))
	return dst
}

func (t *transpiler) transpileUnion(dst []byte, typ *types.Union, mod *Module) []byte {
	if typ.Len() == 0 {
		return append(dst, t.fallbackType...)
	}
	s := set.New[uint64](0)
	seed := maphash.MakeSeed()
	insert := func(b []byte) (modified bool) {
		hash := maphash.Bytes(seed, b)
		if s.Contains(hash) {
			return false
		}
		s.Insert(hash)
		return true
	}
	for term := range typ.Terms() {
		i := len(dst)
		dst = t.transpileType(dst, term.Type(), mod)
		if !insert(dst[i:]) {
			dst = dst[:i]
		} else {
			dst = append(dst, " | "...)
		}
	}
	dst = bytes.TrimSuffix(dst, []byte(" | "))
	return dst
}

func (t *transpiler) transpileTypeParam(dst []byte, typ *types.TypeParam, _ *Module) []byte {
	return append(dst, typ.Obj().Name()...)
}
