// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/internal"
	"github.com/hashicorp/go-set/v3"
	"github.com/invopop/jsonschema"
)

func run() error {
	r := jsonschema.Reflector{
		Anonymous:                  true,
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
	}
	setType := reflect.TypeFor[set.Set[string]]()
	r.Mapper = func(t reflect.Type) *jsonschema.Schema {
		if t == setType {
			schema := r.Reflect([]string(nil))
			schema.Version = ""
			return schema
		}
		return nil
	}
	typ := reflect.TypeFor[internal.Config]()
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
