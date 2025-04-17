/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/internal"
	"github.com/hashicorp/go-set/v3"
	"github.com/invopop/jsonschema"
)

type cmdSchema struct {
	Path string `arg:"" type:"path" default:"tgtt.schema.json"`
}

func (c *cmdSchema) Run() error {
	var f *os.File
	var err error
	if c.Path != "-" {
		if err = os.MkdirAll(filepath.Dir(c.Path), 0750); err != nil {
			return err
		}
		f, err = os.Create(c.Path)
		if err != nil {
			return err
		}
		defer f.Close() //nolint:errcheck
	} else {
		f = os.Stdout
	}

	r := jsonschema.Reflector{
		FieldNameTag:               "yaml",
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
		CommentMap:                 internal.CommentMap(),
	}
	r.Mapper = func(t reflect.Type) *jsonschema.Schema {
		if t == reflect.TypeFor[set.Set[string]]() {
			schema := r.ReflectFromType(reflect.TypeFor[[]string]())
			schema.Version = ""
			return schema
		}
		return nil
	}
	schema := r.ReflectFromType(reflect.TypeFor[internal.Config]())

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	return enc.Encode(schema)
}
