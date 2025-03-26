package tgtt

import (
	"errors"
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

	if err = pkgError(pkgs[0]); err != nil {
		return nil, err
	}

	cache[pattern] = pkgs[0]
	return pkgs[0], nil
}

// Based on [packages.PrintErrors]
func pkgError(pkg *packages.Package) error {
	var sb strings.Builder
	for _, err := range pkg.Errors {
		sb.WriteString(err.Error() + "\n")
	}

	// Print pkg.Module.Error once if present.
	if pkg.Module != nil && pkg.Module.Error != nil {
		sb.WriteString(pkg.Module.Error.Err + "\n")
	}

	if sb.Len() == 0 {
		return nil
	}
	return errors.New(sb.String())
}
