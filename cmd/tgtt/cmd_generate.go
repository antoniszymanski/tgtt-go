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

	var middlewares []tgtt.Middleware
	if cfg.Format {
		middlewares = append(
			middlewares,
			func(b []byte) ([]byte, error) {
				buf, err := sanefmt.Format(bytes.NewReader(b))
				if err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			},
		)
	}

	for _, pkgCfg := range cfg.Packages {
		pkg, err := tgtt.LoadPackage(pkgCfg.Path)
		if err != nil {
			return err
		}

		t := tgtt.NewTranspiler(pkg)
		maps.Copy(t.TypeMappings, cfg.TypeMappings)
		maps.Copy(t.TypeMappings, pkgCfg.TypeMappings)
		t.Transpile(pkgCfg.Names)

		err = t.Index().Generate(pkgCfg.OutputPath, middlewares...)
		if err != nil {
			return err
		}
	}

	return nil
}
