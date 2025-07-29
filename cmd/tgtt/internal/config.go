/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package internal

import "github.com/antoniszymanski/tgtt-go/tgtt"

//go:generate go run ./schemagen

type Config struct {
	Format            bool                  `yaml:"format"`
	IncludeUnexported bool                  `yaml:"include_unexported"`
	OutputPath        string                `yaml:"output_path"`
	TypeMappings      map[string]string     `yaml:"type_mappings"`
	PrimaryPackage    tgtt.PackageOptions   `yaml:"primary_package"`
	SecondaryPackages []tgtt.PackageOptions `yaml:"secondary_packages"`
}
