// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"
	"sync"

	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/internal"
	"github.com/antoniszymanski/tgtt-go/tgtt"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Config struct {
	Schema            string                              `json:"$schema,omitzero"`
	Format            bool                                `json:"format"`
	IncludeUnexported bool                                `json:"include_unexported"`
	FallbackType      string                              `json:"fallback_type" jsonschema:"default=any"`
	OutputPath        string                              `json:"output_path" jsonschema:"required,minLength=1"`
	TypeMappings      internal.Object[string, string]     `json:"type_mappings"`
	PrimaryPackage    tgtt.PackageOptions                 `json:"primary_package" jsonschema:"required"`
	Include           internal.Array[string]              `json:"include"`
	SecondaryPackages internal.Array[tgtt.PackageOptions] `json:"secondary_packages"`
}

func (c *Config) UnmarshalJSON(data []byte) error {
	sch, err := compiledSchema()
	if err != nil {
		return err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if err = sch.Validate(inst); err != nil {
		return err
	}
	type RawConfig Config
	return json.Unmarshal(data, (*RawConfig)(c))
}

var compiledSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schema))
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	if err = compiler.AddResource("memory:", doc); err != nil {
		return nil, err
	}
	return compiler.Compile("memory:")
})

func Schema() string {
	return schema
}

//go:generate go run ../internal/schemagen

//go:embed schema.json
var schema string
