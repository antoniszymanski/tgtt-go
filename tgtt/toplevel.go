package tgtt

import (
	"fmt"
	"go/types"
)

// TODO: name
type topLevel interface {
	Underlying() types.Type
	TypeParams() *types.TypeParamList
	Obj() *types.TypeName
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
