/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package main

import (
	"bytes"
	"io"
	"maps"
	"os"

	"github.com/antoniszymanski/sanefmt-go"
	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/internal"
	"github.com/antoniszymanski/tgtt-go/tgtt"
	"github.com/goccy/go-yaml"
)

type cmdGenerate struct {
	Path string `arg:"" type:"path" default:"tgtt.yml"`
}

func (c *cmdGenerate) Run() error {
	var f *os.File
	var err error
	if c.Path != "-" {
		f, err = os.Open(c.Path)
		if err != nil {
			return err
		}
		defer f.Close() //nolint:errcheck
	} else {
		f = os.Stdin
	}

	var cfg internal.Config
	err = yaml.NewDecoder(f, yaml.UseJSONUnmarshaler()).Decode(&cfg)
	if err != nil && err != io.EOF {
		return err
	}

	var postprocessors []tgtt.PostProcessor
	if cfg.Format {
		postprocessors = append(
			postprocessors,
			func(b []byte) ([]byte, error) {
				b, err := sanefmt.Format(bytes.NewReader(b))
				if err != nil {
					return nil, err
				}
				return b, nil
			},
		)
	}

	for _, pkgCfg := range cfg.Packages {
		var opts []tgtt.TranspilerOption
		{
			typeMappings := make(map[string]string)
			maps.Copy(typeMappings, cfg.TypeMappings)
			maps.Copy(typeMappings, pkgCfg.TypeMappings)
			opts = append(opts, tgtt.TypeMappings(typeMappings))
		}
		if pkgCfg.IncludeUnexported {
			opts = append(opts, tgtt.IncludeUnexported())
		}

		t, err := tgtt.NewTranspiler(pkgCfg.Path, opts...)
		if err != nil {
			return err
		}
		t.Transpile(pkgCfg.Names)

		err = t.Index().WriteTS(pkgCfg.OutputPath, postprocessors...)
		if err != nil {
			return err
		}
	}

	return nil
}
