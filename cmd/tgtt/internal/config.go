/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package internal

import "github.com/hashicorp/go-set/v3"

//go:generate go tool mapcomments-go . . -P internal --mpl2
type Config struct {
	Format       bool              `yaml:"format"`
	TypeMappings map[string]string `yaml:"type_mappings"`
	Packages     []PkgConfig       `yaml:"packages"`
}

type PkgConfig struct {
	Path              string            `yaml:"path"`
	OutputPath        string            `yaml:"output_path"`
	IncludeUnexported bool              `yaml:"include_unexported"`
	Names             *set.Set[string]  `yaml:"names"`
	TypeMappings      map[string]string `yaml:"type_mappings"`
}
