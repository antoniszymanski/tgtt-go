package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/internal"
	"github.com/invopop/jsonschema"
)

type cmdSchema struct {
	Path string `arg:"" type:"path" default:"tgtt.schema.json"`
}

func (c *cmdSchema) Run() error {
	var f *os.File
	var err error
	if c.Path != "-" {
		if err = os.MkdirAll(filepath.Dir(c.Path), 0755); err != nil {
			return err
		}
		f, err = os.Create(c.Path)
		if err != nil {
			return err
		}
		defer f.Close()
	} else {
		f = os.Stdout
	}

	r := jsonschema.Reflector{
		FieldNameTag:               "yaml",
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
		CommentMap:                 internal.CommentMap(),
	}
	schema := r.ReflectFromType(reflect.TypeFor[internal.Config]())

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	return enc.Encode(schema)
}
