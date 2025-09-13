// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/config"
	"github.com/hashicorp/go-set/v3"
)

type cmdInit struct {
	Path       string `arg:"" type:"path" default:"tgtt.jsonc"`
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

	var cfg config.Config
	if !c.NoSchema {
		cfg.Schema = c.SchemaPath
	}
	cfg.PrimaryPackage.Names = set.New[string](0)

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(&cfg)
}
