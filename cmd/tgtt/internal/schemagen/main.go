// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"encoding/json"
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

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(schema); err != nil {
		return err
	}
	data := bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
	return os.WriteFile("schema.json", data, 0600)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
