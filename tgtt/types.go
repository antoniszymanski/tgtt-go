// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import "go/types"

func (t *transpiler) transpileTypeRef(dst []byte, tname *types.TypeName, mod *Module) []byte {
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
			return areObjectsEqual(tname, typ.Obj())
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

func (t *transpiler) transpileTypeArgs(dst []byte, targs *types.TypeList, mod *Module) []byte {
	if targs.Len() == 0 {
		return dst
	}
	dst = append(dst, '<')
	for i := range targs.Len() {
		targ := targs.At(i)
		dst = t.transpileType(dst, targ, mod)
		if i < targs.Len()-1 {
			dst = append(dst, ", "...)
		}
	}
	dst = append(dst, '>')
	return dst
}

func (t *transpiler) transpileTypeParams(dst []byte, tparams *types.TypeParamList, mod *Module) []byte {
	if tparams.Len() == 0 {
		return dst
	}
	dst = append(dst, '<')
	for i := range tparams.Len() {
		tparam := tparams.At(i)
		dst = append(dst, tparam.Obj().Name()...)
		dst = append(dst, " extends "...)
		dst = t.transpileType(dst, tparam.Constraint(), mod)
		if i < tparams.Len()-1 {
			dst = append(dst, ", "...)
		}
	}
	dst = append(dst, '>')
	return dst
}

func qualifiedName(obj types.Object) string {
	if pkg := obj.Pkg(); pkg == nil {
		return obj.Name()
	} else {
		return pkg.Path() + "." + obj.Name()
	}
}

func areObjectsEqual(objA, objB types.Object) bool {
	var pkgPathA, pkgPathB string
	if pkg := objA.Pkg(); pkg != nil {
		pkgPathA = pkg.Path()
	}
	if pkg := objB.Pkg(); pkg != nil {
		pkgPathB = pkg.Path()
	}
	if pkgPathA != pkgPathB {
		return false
	}
	return objA.Name() == objB.Name()
}
