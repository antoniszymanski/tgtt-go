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
	"github.com/vishalkuo/bimap"
	"golang.org/x/tools/go/packages"
)

type transpiler struct {
	pkg          *packages.Package
	pkgNames     *bimap.BiMap[string, string]
	Modules      map[string]*Module
	TypeMappings map[string]string
}

func NewTranspiler(pkg *packages.Package) *transpiler {
	return &transpiler{
		pkg:      pkg,
		pkgNames: bimap.NewBiMap[string, string](),
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

func comment(s string) string {
	return fmt.Sprintf(` /* %s */`, s)
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
			return cmp.Compare(a.Pos(), b.Pos())
		},
	)
	return defs
}

func isTranspilable(obj types.Object) bool {
	if obj == nil {
		return false
	}
	// https://github.com/golang/example/tree/master/gotypes#objects
	switch obj.(type) {
	case *types.Const, *types.TypeName:
		return true
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
		var typStr string
		tobj := typ.Obj()
		pkg := tobj.Pkg()
		if pkg != nil && pkg.Path() != t.pkg.PkgPath {
			typStr = t.include(mod, pkg, tobj.Name())
		} else {
			typStr = obj.Name()
		}

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
		def = t.transpileStructToplevel(typ, mod)
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

func (t *transpiler) transpileBasic(typ *types.Basic, _ *Module) string {
	switch typ.Kind() {
	case types.Bool:
		return "boolean"
	case types.Int:
		return "number" + comment("int")
	case types.Int8:
		return "number" + comment("int8")
	case types.Int16:
		return "number" + comment("int16")
	case types.Int32:
		return "number" + comment("int32")
	case types.Int64:
		return "number" + comment("int64")
	case types.Uint:
		return "number" + comment("uint")
	case types.Uint8:
		return "number" + comment("uint8")
	case types.Uint16:
		return "number" + comment("uint16")
	case types.Uint32:
		return "number" + comment("uint32")
	case types.Uint64:
		return "number" + comment("uint64")
	case types.Uintptr:
		return "number" + comment("uintptr")
	case types.Float32:
		return "number" + comment("float32")
	case types.Float64:
		return "number" + comment("float64")
	case types.String:
		return "string"
	default:
		return "any"
	}
}

func (t *transpiler) transpilePointer(typ *types.Pointer, mod *Module) string {
	return t.transpileType(typ.Elem(), mod)
}

func (t *transpiler) transpileArray(typ *types.Array, mod *Module) string {
	return fmt.Sprintf(`%s[]`, t.transpileType(typ.Elem(), mod))
}

func (t *transpiler) transpileSlice(typ *types.Slice, mod *Module) string {
	return fmt.Sprintf(`%s[]`, t.transpileType(typ.Elem(), mod))
}

func (t *transpiler) transpileMap(typ *types.Map, mod *Module) string {
	return fmt.Sprintf(
		`{ [key in %s]: %s }`,
		t.transpileType(typ.Key(), mod),
		t.transpileType(typ.Elem(), mod),
	)
}

func (t *transpiler) transpileStruct(typ *types.Struct, mod *Module) string {
	var sb strings.Builder
	sb.WriteString("{ ")

	addSpace := true
	for i := range typ.NumFields() {
		field := typ.Field(i)
		if !field.Exported() || field.Embedded() {
			continue
		}

		fieldName, optional := func() (string, bool) {
			tags, err := structtag.Parse(typ.Tag(i))
			if err != nil {
				return "", false
			}

			tag, err := tags.Get("json")
			if err != nil {
				return "", false
			}

			return tag.Name, tag.HasOption("omitempty")
		}()
		if fieldName == "" {
			fieldName = field.Name()
		}
		if fieldName == "-" {
			continue
		}

		format := `%s: %s; `
		if optional {
			format = `%s?: %s; `
		}

		fmt.Fprintf(
			&sb,
			format,
			fieldName,
			t.transpileType(field.Type(), mod),
		)
		addSpace = false
	}

	if addSpace {
		sb.WriteString(" ")
	}
	sb.WriteString("}")

	return sb.String()
}

func (t *transpiler) transpileAlias(typ *types.Alias, mod *Module) string {
	return t.transpileTypeArgs(typ, mod)
}

func (t *transpiler) transpileNamed(typ *types.Named, mod *Module) string {
	if typ.Obj().Pkg() == nil && typ.Obj().Name() == "comparable" {
		return "string | number" + comment("comparable")
	}
	return t.transpileTypeArgs(typ, mod)
}

func (t *transpiler) transpileInterface(typ *types.Interface, mod *Module) string {
	var u *types.Union
	var ok bool
	for i := typ.NumEmbeddeds() - 1; i >= 0; i-- {
		u, ok = typ.EmbeddedType(i).(*types.Union)
		if ok {
			break
		}
	}

	if u == nil {
		return "any"
	}
	return t.transpileUnion(u, mod)
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
