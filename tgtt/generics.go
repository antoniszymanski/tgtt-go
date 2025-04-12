/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package tgtt

import (
	"fmt"
	"go/types"
	"strings"
)

func (t *transpiler) transpileTypeParams(tparams *types.TypeParamList, mod *Module) string {
	var sb strings.Builder

	if tparams.Len() > 0 {
		sb.WriteString("<")
	}
	for i := range tparams.Len() {
		tparam := tparams.At(i)
		fmt.Fprintf(
			&sb,
			`%s extends %s`,
			tparam.Obj().Name(),
			t.transpileType(tparam.Constraint(), mod),
		)
		if i < tparams.Len()-1 {
			sb.WriteString(", ")
		}
	}
	if tparams.Len() > 0 {
		sb.WriteString(">")
	}

	return sb.String()
}

// TODO: name
type typeArgs interface {
	Obj() *types.TypeName
	TypeArgs() *types.TypeList
}

// TODO: name
func (t *transpiler) transpileTypeArgs(typ typeArgs, mod *Module) string {
	var sb strings.Builder

	obj := typ.Obj()
	pkg := obj.Pkg()
	sb.WriteString(t.include(mod, pkg, obj.Name()))

	if typ.TypeArgs().Len() > 0 {
		sb.WriteString("<")
	}
	for i := range typ.TypeArgs().Len() {
		targ := typ.TypeArgs().At(i)
		sb.WriteString(t.transpileType(targ, mod))
		if i < typ.TypeArgs().Len()-1 {
			sb.WriteString(", ")
		}
	}
	if typ.TypeArgs().Len() > 0 {
		sb.WriteString(">")
	}

	return sb.String()
}
