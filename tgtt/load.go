// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import (
	"errors"
	"strings"

	"golang.org/x/tools/go/packages"
)

func loadPackage(pattern string) (*packages.Package, error) {
	// https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns
	// https://pkg.go.dev/golang.org/x/tools/go/packages#pkg-overview
	switch pattern {
	case "main", "pattern=main",
		"all", "pattern=all",
		"std", "pattern=std",
		"cmd", "pattern=cmd",
		"tool", "pattern=tool":
		return nil, errors.New("pattern cannot be a reserved name")
	}
	if strings.Contains(pattern, "...") {
		return nil, errors.New("pattern cannot contain wildcards")
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, err
	}
	pkg := pkgs[0]

	if err = newPackageError(pkg); err != nil {
		return nil, err
	}

	return pkg, nil
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
