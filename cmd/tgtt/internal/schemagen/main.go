// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/antoniszymanski/collections-go/set"
	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/config"
	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/internal"
	"github.com/invopop/jsonschema"
)

func run() error {
	r := jsonschema.Reflector{
		Anonymous:                  true,
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
	}
	r.Mapper = func(t reflect.Type) *jsonschema.Schema {
		if t == reflect.TypeFor[set.Set[string]]() {
			schema := r.ReflectFromType(reflect.TypeFor[[]string]())
			schema.Version = ""
			return schema
		}
		return nil
	}
	typ := reflect.TypeFor[config.Config]()
	if err := r.AddGoComments(typ.PkgPath(), "."); err != nil {
		return err
	}
	schema := r.ReflectFromType(typ)
	data, err := internal.MarshalJSON(schema)
	if err != nil {
		return err
	}
	return os.WriteFile("schema.json", data, 0600)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
