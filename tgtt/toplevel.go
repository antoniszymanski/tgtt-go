package tgtt

import (
	"fmt"
	"go/types"
	"strings"
)

// TODO: name
type topLevel interface {
	Underlying() types.Type
	TypeParams() *types.TypeParamList
	Obj() *types.TypeName
}

func (t *transpiler) transpileStructToplevel(typ topLevel, mod *Module) string {
	tparams := t.transpileTypeParams(typ.TypeParams(), mod)

	path := t.getPkgPath(typ.Obj())
	_, ok := t.TypeMappings[path]
	if ok {
		return fmt.Sprintf(
			`export type %s%s = %s`,
			typ.Obj().Name(), tparams, t.TypeMappings[path],
		)
	}
	typStr := t.transpileStruct(typ.Underlying().(*types.Struct), mod)

	extends := t.transpileExtends(typ.Underlying().(*types.Struct), mod)
	if len(extends) > 0 {
		extends = " " + extends
	}

	return fmt.Sprintf(
		`export interface %s%s%s %s`,
		typ.Obj().Name(), tparams, extends, typStr,
	)
}

func (t *transpiler) transpileExtends(typ *types.Struct, mod *Module) string {
	var extends []string
	for field := range typ.Fields() {
		if !field.Exported() || !field.Embedded() {
			continue
		}

		extends = append(extends, t.transpileType(field.Type(), mod))
	}

	if len(extends) == 0 {
		return ""
	}
	return "extends " + strings.Join(extends, ", ")
}

func (t *transpiler) transpileToplevel(typ topLevel, mod *Module) string {
	tparams := t.transpileTypeParams(typ.TypeParams(), mod)
	var typStr string
	var ok bool
	if typStr, ok = t.TypeMappings[t.getPkgPath(typ.Obj())]; !ok {
		typStr = t.transpileType(typ.Underlying(), mod)
	}

	return fmt.Sprintf(
		`export type %s%s = %s`,
		typ.Obj().Name(), tparams, typStr,
	)
}

func (t *transpiler) getPkgPath(obj types.Object) string {
	var path string
	pkg := obj.Pkg()
	if pkg != nil && pkg.Path() != t.pkg.PkgPath {
		path = pkg.Path() + "."
	}
	path += obj.Name()

	return path
}
