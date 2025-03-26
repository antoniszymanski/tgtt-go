package internal

import "github.com/hashicorp/go-set/v3"

//go:generate go tool mapcomments-go github.com/antoniszymanski/tgtt-go/cmd/tgtt/config . -P internal
type Config struct {
	Format       bool              `yaml:"format"`
	TypeMappings map[string]string `yaml:"type_mappings"`
	Packages     []PkgConfig       `yaml:"packages"`
}

type PkgConfig struct {
	Path         string            `yaml:"path"`
	OutputPath   string            `yaml:"output_path"`
	Names        *set.Set[string]  `yaml:"names"`
	TypeMappings map[string]string `yaml:"type_mappings"`
}
