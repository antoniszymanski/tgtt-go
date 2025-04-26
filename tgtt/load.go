/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package tgtt

import (
	"strings"

	"golang.org/x/tools/go/packages"
)

var cache = make(map[string]*packages.Package)

func LoadPackage(pattern string) (*packages.Package, error) {
	if pkg, ok := cache[pattern]; ok {
		return pkg, nil
	}

	cfg := packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(&cfg, pattern)
	if err != nil {
		return nil, err
	}

	if err = newPackageError(pkgs[0]); err != nil {
		return nil, err
	}

	cache[pattern] = pkgs[0]
	return pkgs[0], nil
}

func newPackageError(pkg *packages.Package) error {
	if len(pkg.Errors) == 0 && (pkg.Module == nil || pkg.Module.Error == nil) {
		return nil
	}

	var err PackageError
	err.Errors = pkg.Errors
	if pkg.Module != nil && pkg.Module.Error != nil {
		err.ModuleError = pkg.Module.Error
	}
	return err
}

type PackageError struct {
	Errors      []packages.Error
	ModuleError *packages.ModuleError
}

// Based on [packages.PrintErrors]
func (err PackageError) Error() string {
	var sb strings.Builder

	for _, pkgErr := range err.Errors {
		sb.WriteString(pkgErr.Error())
		sb.WriteByte('\n')
	}
	if err.ModuleError != nil {
		sb.WriteString(err.ModuleError.Err)
		sb.WriteByte('\n')
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

func (err PackageError) Unwrap() []error {
	pkgErrs := make([]error, 0, len(err.Errors))
	for _, pkgErr := range err.Errors {
		pkgErrs = append(pkgErrs, pkgErr)
	}
	return pkgErrs
}
