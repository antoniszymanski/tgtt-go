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

func (t *transpiler) transpileTypeRef(tname *types.TypeName, mod *TsModule) string {
	if tname.Pkg() == nil {
		switch tname.Name() {
		case "comparable":
			return "string | number /* comparable */"
		case "error":
			return "any /* error */"
		default:
			return tname.Name()
		}
	}

	pkg := t.packages[tname.Pkg().Path()]
	typeMod := t.modules[pkg.Name]
	if typeMod == nil {
		typeMod = t.addModule(pkg.Name, pkg.PkgPath)
	}

	for _, obj := range sortedDefs(pkg) {
		isMatchingConstType := func() bool {
			obj, ok := obj.(*types.Const)
			if !ok {
				return false
			}

			typ, ok := obj.Type().(*types.Named)
			if !ok {
				return false
			}

			return t.getPkgPath(tname) == t.getPkgPath(typ.Obj())
		}()

		if obj.Name() == tname.Name() {
			t.transpileObject(obj, typeMod)
		} else if isMatchingConstType && (obj.Exported() || t.includeUnexported) {
			t.transpileObject(obj, typeMod)
		}
	}

	if typeMod == mod {
		return tname.Name()
	} else {
		mod.Imports.Set(pkg.Name, typeMod)
		return pkg.Name + "." + tname.Name()
	}
}

func (t *transpiler) transpileTypeArgs(targs *types.TypeList, mod *TsModule) string {
	var sb strings.Builder

	if targs.Len() > 0 {
		sb.WriteByte('<')
	}
	for i := range targs.Len() {
		targ := targs.At(i)
		sb.WriteString(t.transpileType(targ, mod))
		if i < targs.Len()-1 {
			sb.WriteString(", ")
		}
	}
	if targs.Len() > 0 {
		sb.WriteByte('>')
	}

	return sb.String()
}

func (t *transpiler) transpileTypeParams(tparams *types.TypeParamList, mod *TsModule) string {
	var sb strings.Builder

	if tparams.Len() > 0 {
		sb.WriteByte('<')
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
		sb.WriteByte('>')
	}

	return sb.String()
}

func (t *transpiler) getPkgPath(obj types.Object) string {
	pkg := obj.Pkg()
	if pkg == nil || pkg.Path() == t.primaryPkg.PkgPath {
		return obj.Name()
	} else {
		return pkg.Path() + "." + obj.Name()
	}
}
