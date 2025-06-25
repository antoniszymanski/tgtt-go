/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package main

import (
	"os"
	"path/filepath"

	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/internal"
	"github.com/goccy/go-yaml"
	"github.com/hashicorp/go-set/v3"
)

type cmdInit struct {
	Path       string `arg:"" type:"path" default:"tgtt.yml"`
	SchemaPath string `arg:"" type:"path" default:"tgtt.schema.json"`
	NoSchema   bool   `short:"S"`
}

func (c *cmdInit) Run() error {
	var f *os.File
	var err error
	if c.Path != "-" {
		dir := filepath.Dir(c.Path)
		if err = os.MkdirAll(dir, 0750); err != nil {
			return err
		}
		f, err = os.Create(c.Path)
		if err != nil {
			return err
		}
		defer f.Close() //nolint:errcheck

		if !c.NoSchema {
			relpath, err := filepath.Rel(dir, c.SchemaPath)
			if err == nil {
				c.SchemaPath = relpath
			}
		}
	} else {
		f = os.Stdout
	}

	if !c.NoSchema {
		s := "# yaml-language-server: $schema=" + c.SchemaPath + "\n"
		_, err = f.WriteString(s)
		if err != nil {
			return err
		}
	}
	var cfg internal.Config
	cfg.PrimaryPackage.Names = set.New[string](0)
	return yaml.NewEncoder(f, yaml.UseJSONMarshaler()).Encode(cfg)
}
