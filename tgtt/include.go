package tgtt

import (
	"fmt"
	"go/types"
	"strconv"
)

func (t *transpiler) include(root *Module, tpkg *types.Package, objName string) string {
	pkgName := t.getPkgName(tpkg)
	var mod *Module
	var ok bool
	if root.GoPath == tpkg.Path() {
		mod = root
	} else {
		mod, ok = root.Imports.Get(pkgName)
		if !ok {
			mod = t.getModule(pkgName, tpkg.Path())
			root.Imports.Set(pkgName, mod)
		}
	}

	ret := func() string {
		if mod == root {
			return objName
		}
		return pkgName + "." + objName
	}
	if mod.Defs.Has(objName) {
		return ret()
	}

	pkg := t.pkg.Imports[tpkg.Path()]
	if pkg == nil {
		var err error
		pkg, err = LoadPackage(tpkg.Path())
		if err != nil {
			mod.Defs.Set(
				objName, fmt.Sprintf(`export type %s = any`, objName),
			)
			return ret()
		}
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

			return tpkg.Path()+"."+objName == t.getPkgPath(typ.Obj())
		}()

		if obj.Name() == objName || isMatchingConstType {
			t.transpileObject(obj, mod)
		}
	}

	return ret()
}

func (t *transpiler) getPkgName(pkg *types.Package) string {
	if name, ok := t.pkgNames.Lookup(pkg.Path()); ok {
		return name
	}

	nameExists := func(name string) bool {
		return t.pkgNames.HasInverse(name) ||
			t.pkg.Types.Scope().Lookup(name) != nil ||
			t.pkg.Types.Name() == name
	}
	name := pkg.Name()
	for i := uint64(1); nameExists(name); i++ {
		name = pkg.Name() + strconv.FormatUint(i, 10)
	}
	t.pkgNames.Set(pkg.Path(), name)
	return name
}

func (t *transpiler) getModule(name, goPath string) *Module {
	if mod, ok := t.Modules[name]; ok {
		return mod
	}

	mod := NewModule(goPath)
	t.Modules[name] = mod
	return mod
}
