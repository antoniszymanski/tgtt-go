/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

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
	path := t.getPkgPath(typ.Obj())
	typStr, ok := t.TypeMappings[path]
	if !ok {
		typStr = t.transpileType(typ.Underlying(), mod)
	}

	return fmt.Sprintf(
		`export type %s%s = %s`,
		typ.Obj().Name(),
		t.transpileTypeParams(typ.TypeParams(), mod),
		typStr,
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
