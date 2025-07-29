// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

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
