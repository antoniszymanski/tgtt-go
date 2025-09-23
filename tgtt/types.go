// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import "go/types"

func (t *transpiler) transpileTypeRef(dst []byte, tname *types.TypeName, mod *TsModule) []byte {
	if tname.Pkg() == nil {
		switch tname.Name() {
		case "comparable":
			return append(dst, "string | number /* comparable */"...)
		case "error":
			return append(dst, "any /* error */"...)
		default:
			return append(dst, tname.Name()...)
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
		dst = append(dst, tname.Name()...)
	} else {
		mod.Imports.Set(pkg.Name, typeMod)
		dst = append(dst, pkg.Name...)
		dst = append(dst, '.')
		dst = append(dst, tname.Name()...)
	}
	return dst
}

func (t *transpiler) transpileTypeArgs(dst []byte, targs *types.TypeList, mod *TsModule) []byte {
	if targs.Len() > 0 {
		dst = append(dst, '<')
	}
	for i := range targs.Len() {
		targ := targs.At(i)
		dst = t.transpileType(dst, targ, mod)
		if i < targs.Len()-1 {
			dst = append(dst, ", "...)
		}
	}
	if targs.Len() > 0 {
		dst = append(dst, '>')
	}
	return dst
}

func (t *transpiler) transpileTypeParams(dst []byte, tparams *types.TypeParamList, mod *TsModule) []byte {
	if tparams.Len() > 0 {
		dst = append(dst, '<')
	}
	for i := range tparams.Len() {
		tparam := tparams.At(i)
		dst = append(dst, tparam.Obj().Name()...)
		dst = append(dst, " extends "...)
		dst = t.transpileType(dst, tparam.Constraint(), mod)
		if i < tparams.Len()-1 {
			dst = append(dst, ", "...)
		}
	}
	if tparams.Len() > 0 {
		dst = append(dst, '>')
	}
	return dst
}

func (t *transpiler) getPkgPath(obj types.Object) string {
	pkg := obj.Pkg()
	if pkg == nil || pkg.Path() == t.primaryPkg.PkgPath {
		return obj.Name()
	} else {
		return pkg.Path() + "." + obj.Name()
	}
}
